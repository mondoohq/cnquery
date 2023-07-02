package ssh

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"errors"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/ssh/awsinstanceconnect"
	"go.mondoo.com/cnquery/motor/providers/ssh/awsssmsession"
	"go.mondoo.com/cnquery/motor/providers/ssh/signers"
	"go.mondoo.com/cnquery/motor/vault"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func establishClientConnection(pCfg *providers.Config, hostKeyCallback ssh.HostKeyCallback) (*ssh.Client, []io.Closer, error) {
	authMethods, closer, err := prepareConnection(pCfg)
	if err != nil {
		return nil, nil, err
	}

	if len(authMethods) == 0 {
		return nil, nil, errors.New("no authentication method defined")
	}

	// TODO: hack: we want to establish a proper connection per configured connection so that we could use multiple users
	user := ""
	for i := range pCfg.Credentials {
		if pCfg.Credentials[i].User != "" {
			user = pCfg.Credentials[i].User
		}
	}

	log.Debug().Int("methods", len(authMethods)).Str("user", user).Msg("connect to remote ssh")
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", pCfg.Host, pCfg.Port), &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	})
	return conn, closer, err
}

// hasAgentLoadedKey returns if the ssh agent has loaded the key file
// This may not be 100% accurate. The key can be stored in multiple locations with the
// same fingerprint. We cannot determine the fingerprint without decoding the encrypted
// key, `ssh-keygen -lf /Users/chartmann/.ssh/id_rsa` seems to use the ssh agent to
// determine the fingerprint without prompting for the password
func hasAgentLoadedKey(list []*agent.Key, filename string) bool {
	for i := range list {
		if list[i].Comment == filename {
			return true
		}
	}
	return false
}

// prepareConnection determines the auth methods required for a ssh connection and also prepares any other
// pre-conditions for the connection like tunnelling the connection via AWS SSM session
func prepareConnection(pCfg *providers.Config) ([]ssh.AuthMethod, []io.Closer, error) {
	auths := []ssh.AuthMethod{}
	closer := []io.Closer{}

	// only one public auth method is allowed, therefore multiple keys need to be encapsulated into one auth method
	sshSigners := []ssh.Signer{}

	// use key auth, only load if the key was not found in ssh agent
	for i := range pCfg.Credentials {
		credential := pCfg.Credentials[i]

		switch credential.Type {
		case vault.CredentialType_private_key:
			log.Debug().Msg("enabled ssh private key authentication")
			priv, err := signers.GetSignerFromPrivateKeyWithPassphrase(credential.Secret, []byte(credential.Password))
			if err != nil {
				log.Debug().Err(err).Msg("could not read private key")
			} else {
				sshSigners = append(sshSigners, priv)
			}
		case vault.CredentialType_password:
			// use password auth if the password was set, this is also used when only the username is set
			if len(credential.Secret) > 0 {
				log.Debug().Msg("enabled ssh password authentication")
				auths = append(auths, ssh.Password(string(credential.Secret)))
			}
		case vault.CredentialType_ssh_agent:
			log.Debug().Msg("enabled ssh agent authentication")
			sshSigners = append(sshSigners, signers.GetSignersFromSSHAgent()...)
		case vault.CredentialType_aws_ec2_ssm_session:
			// when the user establishes the ssm session we do the following
			// 1. start websocket connection and start the session-manager-plugin to map the websocket to a local port
			// 2. create new ssh key via instance connect so that we do not rely on any pre-existing ssh key
			err := awsssmsession.CheckPlugin()
			if err != nil {
				return nil, nil, errors.New("Local AWS Session Manager plugin is missing. See https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html for information on the AWS Session Manager plugin and installation instructions")
			}

			loadOpts := []func(*config.LoadOptions) error{}
			if pCfg.Options != nil && pCfg.Options["region"] != "" {
				loadOpts = append(loadOpts, config.WithRegion(pCfg.Options["region"]))
			}
			profile := ""
			if pCfg.Options != nil && pCfg.Options["profile"] != "" {
				loadOpts = append(loadOpts, config.WithSharedConfigProfile(pCfg.Options["profile"]))
				profile = pCfg.Options["profile"]
			}
			log.Debug().Str("profile", pCfg.Options["profile"]).Str("region", pCfg.Options["region"]).Msg("using aws creds")

			cfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
			if err != nil {
				return nil, nil, err
			}

			// we use ec2 instance connect api to create credentials for an aws instance
			eic := awsinstanceconnect.New(cfg)
			host := pCfg.Host
			if id, ok := pCfg.Options["instance"]; ok {
				host = id
			}
			creds, err := eic.GenerateCredentials(host, credential.User)
			if err != nil {
				return nil, nil, err
			}

			// we use ssm session manager to connect to instance via websockets
			sManager, err := awsssmsession.NewAwsSsmSessionManager(cfg, profile)
			if err != nil {
				return nil, nil, err
			}

			// prepare websocket connection and bind it to a free local port
			localIp := "localhost"
			remotePort := "22"
			// NOTE: for SSM we always target the instance id
			pCfg.Host = creds.InstanceId
			localPort, err := awsssmsession.GetAvailablePort()
			if err != nil {
				return nil, nil, errors.New("could not find an available port to start the ssm proxy")
			}
			ssmConn, err := sManager.Dial(pCfg, strconv.Itoa(localPort), remotePort)
			if err != nil {
				return nil, nil, err
			}

			// update endpoint information for ssh to connect via local ssm proxy
			// TODO: this has a side-effect, we may need extend the struct to include resolved connection data
			pCfg.Host = localIp
			pCfg.Port = int32(localPort)

			// NOTE: we need to set insecure so that ssh does not complain about the host key
			// It is okay do that since the connection is established via aws api itself and it ensures that
			// the instance id is okay
			pCfg.Insecure = true

			// use the generated ssh credentials for authentication
			priv, err := signers.GetSignerFromPrivateKeyWithPassphrase(creds.KeyPair.PrivateKey, creds.KeyPair.Passphrase)
			if err != nil {
				return nil, nil, errors.Join(err, errors.New("could not read generated private key"))
			}
			sshSigners = append(sshSigners, priv)
			closer = append(closer, ssmConn)
		case vault.CredentialType_aws_ec2_instance_connect:
			log.Debug().Str("profile", pCfg.Options["profile"]).Str("region", pCfg.Options["region"]).Msg("using aws creds")

			loadOpts := []func(*config.LoadOptions) error{}
			if pCfg.Options != nil && pCfg.Options["region"] != "" {
				loadOpts = append(loadOpts, config.WithRegion(pCfg.Options["region"]))
			}
			if pCfg.Options != nil && pCfg.Options["profile"] != "" {
				loadOpts = append(loadOpts, config.WithSharedConfigProfile(pCfg.Options["profile"]))
			}
			cfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
			if err != nil {
				return nil, nil, err
			}
			log.Debug().Msg("generating instance connect credentials")
			eic := awsinstanceconnect.New(cfg)
			host := pCfg.Host
			if id, ok := pCfg.Options["instance"]; ok {
				host = id
			}
			creds, err := eic.GenerateCredentials(host, credential.User)
			if err != nil {
				return nil, nil, err
			}

			priv, err := signers.GetSignerFromPrivateKeyWithPassphrase(creds.KeyPair.PrivateKey, creds.KeyPair.Passphrase)
			if err != nil {
				return nil, nil, errors.Join(err, errors.New("could not read generated private key"))
			}
			sshSigners = append(sshSigners, priv)

			// NOTE: this creates a side-effect where the host is overwritten
			pCfg.Host = creds.PublicIpAddress
		default:
			return nil, nil, errors.New("unsupported authentication mechanism for ssh: " + credential.Type.String())
		}
	}

	if len(sshSigners) > 0 {
		auths = append(auths, ssh.PublicKeys(sshSigners...))
	}

	// if no credential was provided, fallback to ssh-agent and ssh-config
	if len(pCfg.Credentials) == 0 {
		sshSigners = append(sshSigners, signers.GetSignersFromSSHAgent()...)
	}

	return auths, closer, nil
}

package ssh

import (
	"context"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/ssh/awsinstanceconnect"
	"go.mondoo.io/mondoo/motor/transports/ssh/awsssmsession"
	"go.mondoo.io/mondoo/motor/vault"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func establishClientConnection(tc *transports.TransportConfig, hostKeyCallback ssh.HostKeyCallback) (*ssh.Client, []io.Closer, error) {
	authMethods, closer, err := prepareConnection(tc)
	if err != nil {
		return nil, nil, err
	}

	log.Debug().Int("methods", len(authMethods)).Msg("discovered ssh auth methods")
	if len(authMethods) == 0 {
		return nil, nil, errors.New("no authentication method defined")
	}

	// TODO: hack: we want to establish a proper connection per configured connection so that we could use multiple users
	user := ""
	for i := range tc.Credentials {
		if tc.Credentials[i].User != "" {
			user = tc.Credentials[i].User
		}
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", tc.Host, tc.Port), &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	})
	return conn, closer, err
}

func authPrivateKeyWithPassphrase(pemBytes []byte, passphrase []byte) (ssh.Signer, error) {
	log.Debug().Msg("enabled ssh private key authentication")

	// check if the key is encrypted
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("ssh: no key found")
	}

	var signer ssh.Signer
	var err error
	if strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
		// we may want to support to parse password protected encrypted key
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, passphrase)
		if err != nil {
			return nil, err
		}
	} else {
		// parse unencrypted key
		signer, err = ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, err
		}
	}

	return signer, nil
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
func prepareConnection(tc *transports.TransportConfig) ([]ssh.AuthMethod, []io.Closer, error) {
	auths := []ssh.AuthMethod{}
	closer := []io.Closer{}

	// only one public auth method is allowed, therefore multiple keys need to be encapsulated into one auth method
	signers := []ssh.Signer{}

	// enable ssh agent auth
	useAgentAuth := func() {
		if sshAgentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			log.Debug().Str("socket", os.Getenv("SSH_AUTH_SOCK")).Msg("enabled ssh agent authentication")
			sshAgentClient := agent.NewClient(sshAgentConn)
			sshAgentSigners, err := sshAgentClient.Signers()
			if err == nil && len(sshAgentSigners) == 0 {
				log.Warn().Msg("could not find keys in ssh agent")
			} else if err == nil {
				signers = append(signers, sshAgentSigners...)
			} else {
				log.Error().Err(err).Msg("could not get public keys from ssh agent")
			}
		} else {
			log.Debug().Msg("could not find valud ssh agent authentication")
		}
	}

	// use key auth, only load if the key was not found in ssh agent
	for i := range tc.Credentials {
		credential := tc.Credentials[i]

		switch credential.Type {
		case vault.CredentialType_private_key:
			log.Debug().Msg("enabled ssh private key authentication")
			priv, err := authPrivateKeyWithPassphrase(credential.Secret, []byte(credential.Password))
			if err != nil {
				log.Debug().Err(err).Msg("could not read private key")
			} else {
				signers = append(signers, priv)
			}
		case vault.CredentialType_password:
			// use password auth
			log.Debug().Msg("enabled ssh password authentication")
			auths = append(auths, ssh.Password(string(credential.Secret)))
		case vault.CredentialType_ssh_agent:
			log.Debug().Msg("enabled ssh agent authentication")
			useAgentAuth()
		case vault.CredentialType_aws_ec2_ssm_session:
			// when the user establishes the ssm session we do the following
			// 1. start websocket connection and start the session-manager-plugin to map the websocket to a local port
			// 2. create new ssh key via instance connect so that we do not rely on any pre-existing ssh key
			err := awsssmsession.CheckPlugin()
			if err != nil {
				return nil, nil, errors.New("AWS Session Manager plugin is missing. More information at https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html")
			}

			loadOpts := []func(*config.LoadOptions) error{}
			if tc.Options != nil && tc.Options["region"] != "" {
				loadOpts = append(loadOpts, config.WithRegion(tc.Options["region"]))
			}
			profile := ""
			if tc.Options != nil && tc.Options["profile"] != "" {
				loadOpts = append(loadOpts, config.WithSharedConfigProfile(tc.Options["profile"]))
				profile = tc.Options["profile"]
			}

			cfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
			if err != nil {
				return nil, nil, err
			}

			// we use ec2 instance connect api to create credentials for an aws instance
			eic := awsinstanceconnect.New(cfg)
			creds, err := eic.GenerateCredentials(tc.Host, credential.User)
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

			localPort, err := awsssmsession.GetAvailablePort()
			if err != nil {
				return nil, nil, errors.New("could not find an available port to start the ssm proxy")
			}
			ssmConn, err := sManager.Dial(tc, strconv.Itoa(localPort), remotePort)
			if err != nil {
				return nil, nil, err
			}

			// update endpoint information for ssh to connect via local ssm proxy
			// TODO: this has a side-effect, we may need extend the struct to include resolved connection data
			tc.Host = localIp
			tc.Port = int32(localPort)

			// NOTE: we need to set insecure so that ssh does not complain about the host key
			// It is okay do that since the connection is established via aws api itself and it ensures that
			// the instance id is okay
			tc.Insecure = true

			// use the generated ssh credentials for authentication
			priv, err := authPrivateKeyWithPassphrase(creds.KeyPair.PrivateKey, creds.KeyPair.Passphrase)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not read generated private key")
			}
			signers = append(signers, priv)
			closer = append(closer, ssmConn)
		case vault.CredentialType_aws_ec2_instance_connect:
			cfg, err := config.LoadDefaultConfig(context.Background())
			if err != nil {
				return nil, nil, err
			}

			eic := awsinstanceconnect.New(cfg)
			creds, err := eic.GenerateCredentials(tc.Host, credential.User)
			if err != nil {
				return nil, nil, err
			}

			priv, err := authPrivateKeyWithPassphrase(creds.KeyPair.PrivateKey, creds.KeyPair.Passphrase)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not read generated private key")
			}
			signers = append(signers, priv)

			// NOTE: this creates a side-effect where the host is overwritten
			tc.Host = creds.PublicIpAddress
		default:
			return nil, nil, errors.New("unsupported authentication mechanism for ssh: " + credential.Type.String())
		}
	}

	// if no credential was provided, fallback to ssh-agent and ssh-config
	if len(tc.Credentials) == 0 {
		useAgentAuth()
	}

	if len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}
	return auths, closer, nil
}

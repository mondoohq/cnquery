package ssh

import (
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"go.mondoo.io/mondoo/motor/vault"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshClientConnection(cc *transports.TransportConfig, hostKeyCallback ssh.HostKeyCallback) (*ssh.Client, error) {
	authMethods, err := authMethods(cc)
	if err != nil {
		return nil, err
	}

	log.Debug().Int("methods", len(authMethods)).Msg("discovered ssh auth methods")
	if len(authMethods) == 0 {
		return nil, errors.New("no authentication method defined")
	}

	// TODO: hack: we want to establish a proper connection per configured connection so that we could use multiple users
	user := ""
	for i := range cc.Credentials {
		if cc.Credentials[i].User != "" {
			user = cc.Credentials[i].User
		}
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%s", cc.Host, cc.Port), sshConfig)
}

func authPrivateKeyWithPassphrase(pemBytes []byte, password string) (ssh.Signer, error) {
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
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(password))
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

func authMethods(tc *transports.TransportConfig) ([]ssh.AuthMethod, error) {
	auths := []ssh.AuthMethod{}

	// only one public auth method is allowed, therefore multiple keys need to be encapsulated into one auth method
	signers := []ssh.Signer{}

	// enable ssh agent auth
	useAgentAuth := func() {
		if sshAgentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			log.Debug().Str("socket", os.Getenv("SSH_AUTH_SOCK")).Msg("enabled ssh agent authentication")
			sshAgentClient := agent.NewClient(sshAgentConn)
			sshAgentSigners, err := sshAgentClient.Signers()
			if err == nil {
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
			priv, err := authPrivateKeyWithPassphrase(credential.Secret, credential.Password)
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
		default:
			return nil, errors.New("unsupported authentication mechanism for ssh: " + credential.Type.String())
		}
	}

	// if no credential was provided, fallback to ssh-agent and ssh-config
	if len(tc.Credentials) == 0 {
		useAgentAuth()
	}

	if len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}
	return auths, nil
}

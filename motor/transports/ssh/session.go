package ssh

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshClientConnection(hostconfig *transports.TransportConfig, hostKeyCallback ssh.HostKeyCallback) (*ssh.Client, error) {
	authMethods, err := authMethods(hostconfig)
	if err != nil {
		return nil, err
	}

	log.Debug().Int("methods", len(authMethods)).Msg("discovered ssh auth methods")
	if len(authMethods) == 0 {
		return nil, errors.New("no authentication method defined")
	}

	sshConfig := &ssh.ClientConfig{
		User:            hostconfig.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%s", hostconfig.Host, hostconfig.Port), sshConfig)
}

func authPrivateKey(privateKeyPath string, password string) (ssh.Signer, error) {
	log.Debug().Str("key", privateKeyPath).Msg("enabled ssh private key authentication")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return nil, errors.New("private key does not exist " + privateKeyPath)
	}

	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	// check if the key is encrypted
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("ssh: no key found")
	}

	var signer ssh.Signer
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

func authMethods(endpoint *transports.TransportConfig) ([]ssh.AuthMethod, error) {
	auths := []ssh.AuthMethod{}

	// only one public auth method is allowed, therefore multiple keys need to be encapsulated into one auth method
	signers := []ssh.Signer{}

	// enable ssh agent auth
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

	// use key auth, only load if the key was not found in ssh agent
	for i := range endpoint.IdentityFiles {
		identityKey := endpoint.IdentityFiles[i]
		if len(identityKey) == 0 {
			continue
		}
		priv, err := authPrivateKey(identityKey, endpoint.Password)
		if err != nil {
			log.Debug().Err(err).Str("key", identityKey).Msg("could not load private key, ignore the file")
		} else {
			signers = append(signers, priv)
		}
	}

	if len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}

	// use password auth
	if endpoint.Password != "" {
		log.Debug().Msg("enabled ssh password authentication")
		auths = append(auths, ssh.Password(endpoint.Password))
	}
	return auths, nil
}

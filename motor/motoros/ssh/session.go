package ssh

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"net"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshClientConnection(hostconfig *types.Endpoint, hostKeyCallback ssh.HostKeyCallback) (*ssh.Client, error) {
	authMethods, err := authMethods(hostconfig)
	if err != nil {
		return nil, err
	}

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

func authPrivateKey(privateKeyPath string, password string) (ssh.AuthMethod, error) {
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

	return ssh.PublicKeys(signer), nil
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

func authMethods(endpoint *types.Endpoint) ([]ssh.AuthMethod, error) {
	auths := []ssh.AuthMethod{}

	sshAgentKeys := []*agent.Key{}

	// enable ssh agent auth
	if sshAgentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		log.Debug().Str("socket", os.Getenv("SSH_AUTH_SOCK")).Msg("enabled ssh agent authentication")
		sshAgentClient := agent.NewClient(sshAgentConn)
		agentAuth := ssh.PublicKeysCallback(sshAgentClient.Signers)
		auths = append(auths, agentAuth)

		// includes all loaded keys
		list, err := sshAgentClient.List()
		if err == nil {
			sshAgentKeys = list
		}
	} else {
		log.Debug().Msg("could not find valud ssh agent authentication")
	}

	// use key auth, only load if the key was not found in ssh agent
	if endpoint.PrivateKeyPath != "" && !hasAgentLoadedKey(sshAgentKeys, endpoint.PrivateKeyPath) {
		priv, err := authPrivateKey(endpoint.PrivateKeyPath, endpoint.Password)
		if err != nil {
			log.Warn().Err(err).Str("key", endpoint.PrivateKeyPath).Msg("could not load private key, fallback to ssh agent")
		} else {
			auths = append(auths, priv)
		}
	}

	// use password auth
	if endpoint.Password != "" {
		log.Debug().Msg("enabled ssh password authentication")
		auths = append(auths, ssh.Password(endpoint.Password))
	}

	return auths, nil
}

package ssh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"net"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
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

func authMethods(endpoint *types.Endpoint) ([]ssh.AuthMethod, error) {
	auths := []ssh.AuthMethod{}

	if endpoint.PrivateKeyPath != "" {
		log.Debug().Str("key", endpoint.PrivateKeyPath).Msg("load private key")
		if _, err := os.Stat(endpoint.PrivateKeyPath); os.IsNotExist(err) {
			return auths, errors.New("private key does not exist " + endpoint.PrivateKeyPath)
		}

		pemBytes, err := ioutil.ReadFile(endpoint.PrivateKeyPath)
		if err != nil {
			return auths, err
		}

		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return auths, err
		}

		auths = append(auths, ssh.PublicKeys(signer))
	}

	if endpoint.Password != "" {
		auths = append(auths, ssh.Password(endpoint.Password))
	}

	agentAuth := sshAgent()
	if agentAuth != nil {
		auths = append(auths, agentAuth)
	}

	return auths, nil
}

func sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

package ssh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
	"golang.org/x/crypto/ssh"
)

func sshClient(hostconfig *types.Endpoint) (*ssh.Client, error) {
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", hostconfig.Host, hostconfig.Port), sshConfig)
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

	return auths, nil
}

func sftpClient(sshClient *ssh.Client) (*sftp.Client, error) {
	c, err := sftp.NewClient(sshClient, sftp.MaxPacket(1<<15))
	if err != nil {
		return nil, err
	}
	return c, nil
}

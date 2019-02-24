package ssh

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
	"golang.org/x/crypto/ssh"
)

func VerifyConfig(endpoint *types.Endpoint) error {
	if endpoint.Backend != "ssh" {
		return errors.New("only ssh backend for ssh transport supported")
	}

	_, err := endpoint.IntPort()
	if err != nil {
		return errors.New("port is not a valid number " + endpoint.Port)
	}

	return nil
}

func DefaultConfig(endpoint *types.Endpoint) *types.Endpoint {
	p, err := endpoint.IntPort()
	// use default port if port is 0
	if err == nil && p <= 0 {
		endpoint.Port = "22"
	}
	return endpoint
}

func New(endpoint *types.Endpoint) (*SSHTransport, error) {
	// ensure all required configs are set
	err := VerifyConfig(endpoint)
	if err != nil {
		return nil, err
	}

	// set default config if required
	endpoint = DefaultConfig(endpoint)

	// establish connection
	conn, err := sshClient(endpoint)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("transport", "ssh").Msg("session established")
	return &SSHTransport{Endpoint: endpoint, SSHClient: conn}, nil
}

type SSHTransport struct {
	Endpoint  *types.Endpoint
	SSHClient *ssh.Client
}

func (t *SSHTransport) RunCommand(command string) (*types.Command, error) {
	log.Debug().Str("command", command).Str("transport", "ssh").Msg("run command")
	c := &Command{SSHClient: t.SSHClient}
	return c.Exec(command)
}

func (t *SSHTransport) File(path string) (types.File, error) {
	log.Debug().Str("path", path).Str("transport", "ssh").Msg("fetch file")
	f := &File{SSHClient: t.SSHClient, filePath: path}
	return f, nil
}

func (t *SSHTransport) Close() {
	if t.SSHClient != nil {
		t.SSHClient.Close()
	}
}

package ssh

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
	"golang.org/x/crypto/ssh"
)

func New(endpoint *types.Endpoint) (*SSHTransport, error) {
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

package ssh

import (
	"net"
	"os"

	"github.com/pkg/errors"

	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/motoros/ssh/scp"
	"go.mondoo.io/mondoo/motor/motoros/ssh/sftp"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"golang.org/x/crypto/ssh"
)

func ReadSSHConfig(endpoint *types.Endpoint) *types.Endpoint {
	// optional step, tries to parse the ssh config to see if additional information
	// is already available
	if len(endpoint.User) == 0 {
		endpoint.User = ssh_config.Get(endpoint.Host, "User")
	}

	if len(endpoint.Port) == 0 {
		endpoint.Port = ssh_config.Get(endpoint.Host, "Port")
	}

	if len(endpoint.PrivateKeyPath) == 0 {
		entry := ssh_config.Get(endpoint.Host, "IdentityFile")
		// TODO: the ssh_config uses os/home but instead should be use go-homedir, could become a compile issue
		// TODO: the problem is that the lib returns defaults and we cannot properly distingush
		if ssh_config.Default("IdentityFile") != entry {
			// commonly ssh config included paths like ~
			expanded, err := homedir.Expand(entry)
			if err == nil {
				log.Debug().Str("key", expanded).Str("host", endpoint.Host).Msg("read ssh identity key from ssh config")
				endpoint.PrivateKeyPath = expanded
			}
		}
	}

	return endpoint
}

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
	endpoint = ReadSSHConfig(endpoint)

	// ensure all required configs are set
	err := VerifyConfig(endpoint)
	if err != nil {
		return nil, err
	}

	// set default config if required
	endpoint = DefaultConfig(endpoint)

	// establish connection
	var hostkey ssh.PublicKey
	conn, err := sshClientConnection(endpoint, func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hostkey = key
		// TODO: we may want to be more strict here
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Debug().Str("transport", "ssh").Msg("session established")

	activateScp := false
	if os.Getenv("MONDOO_SSH_SCP") == "on" {
		activateScp = true
		log.Info().Str("transport", "ssh").Msg("ssh uses scp (beta) instead of sftp for file transfer")
	}

	return &SSHTransport{
		Endpoint:             endpoint,
		SSHClient:            conn,
		UseBetaScpFilesystem: activateScp,
		HostKey:              hostkey,
	}, nil
}

type SSHTransport struct {
	Endpoint             *types.Endpoint
	SSHClient            *ssh.Client
	fs                   afero.Fs
	UseBetaScpFilesystem bool
	HostKey              ssh.PublicKey
}

func (t *SSHTransport) RunCommand(command string) (*types.Command, error) {
	log.Debug().Str("command", command).Str("transport", "ssh").Msg("run command")
	c := &Command{SSHClient: t.SSHClient}
	return c.Exec(command)
}

func (t *SSHTransport) FS() afero.Fs {
	if t.fs == nil {
		if t.UseBetaScpFilesystem {
			t.fs = scp.NewFs(t.SSHClient)
		} else {
			fs, err := sftp.New(t.SSHClient)
			if err != nil {
				log.Error().Err(err).Msg("error during sftp initialization")
			} else {
				t.fs = fs
			}
		}
	}
	return t.fs
}

func (t *SSHTransport) File(path string) (afero.File, error) {
	fs := t.FS()
	if fs == nil {
		return nil, errors.New("could not initialize the ssh filesystem")
	}

	return fs.Open(path)
}

func (t *SSHTransport) Close() {
	if t.SSHClient != nil {
		t.SSHClient.Close()
	}
}

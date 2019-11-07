package ssh

import (
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/motor/motoros/ssh/scp"
	"go.mondoo.io/mondoo/motor/motoros/ssh/sftp"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"golang.org/x/crypto/ssh"
)

func ReadSSHConfig(endpoint *types.Endpoint) *types.Endpoint {
	host := endpoint.Host

	home, err := homedir.Dir()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to determine user home directory")
		return endpoint
	}

	sshUserConfigPath := filepath.Join(home, ".ssh", "config")
	f, err := os.Open(sshUserConfigPath)
	if err != nil {
		log.Debug().Err(err).Str("file", sshUserConfigPath).Msg("Could not read ssh config")
		return endpoint
	}

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		log.Debug().Err(err).Str("file", sshUserConfigPath).Msg("Could not parse ssh config")
		return endpoint
	}

	// optional step, tries to parse the ssh config to see if additional information
	// is already available
	hostname, err := cfg.Get(host, "HostName")
	if err == nil && len(hostname) > 0 {
		endpoint.Host = hostname
	}

	if len(endpoint.User) == 0 {
		user, err := cfg.Get(host, "User")
		if err == nil {
			endpoint.User = user
		}
	}

	if len(endpoint.Port) == 0 {
		port, err := cfg.Get(host, "Port")
		if err == nil {
			endpoint.Port = port
		}
	}

	if len(endpoint.PrivateKeyPath) == 0 {
		entry, err := cfg.Get(host, "IdentityFile")
		// TODO: the ssh_config uses os/home but instead should be use go-homedir, could become a compile issue
		// TODO: the problem is that the lib returns defaults and we cannot properly distingush
		if err == nil && ssh_config.Default("IdentityFile") != entry {
			// commonly ssh config included paths like ~
			expanded, err := homedir.Expand(entry)
			if err == nil {
				log.Debug().Str("key", expanded).Str("host", host).Msg("read ssh identity key from ssh config")
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
		log.Warn().Err(err).Str("transport", "ssh").Str("host", endpoint.Host).Str("port", endpoint.Port).Str("user", endpoint.User).Msg("could not establish ssh session")
		return nil, err
	}

	log.Debug().Str("transport", "ssh").Str("host", endpoint.Host).Str("port", endpoint.Port).Str("user", endpoint.User).Msg("ssh session established")
	activateScp := false
	if os.Getenv("MONDOO_SSH_SCP") == "on" {
		activateScp = true
	}

	return &SSHTransport{
		Endpoint:         endpoint,
		SSHClient:        conn,
		UseScpFilesystem: activateScp,
		HostKey:          hostkey,
	}, nil
}

type SSHTransport struct {
	Endpoint         *types.Endpoint
	SSHClient        *ssh.Client
	fs               afero.Fs
	UseScpFilesystem bool
	HostKey          ssh.PublicKey
}

func (t *SSHTransport) RunCommand(command string) (*types.Command, error) {
	log.Debug().Str("command", command).Str("transport", "ssh").Msg("run command")
	c := &Command{SSHClient: t.SSHClient}
	return c.Exec(command)
}

func (t *SSHTransport) FS() afero.Fs {
	if t.fs == nil {
		// we always try to use sftp first (if scp is not user-enforced)
		// and we also fallback to scp if sftp does not work
		if !t.UseScpFilesystem {
			fs, err := sftp.New(t.SSHClient)
			if err != nil {
				log.Error().Err(err).Msg("error during sftp initialization, enable fallback to scp")
				// enable fallback
				t.UseScpFilesystem = true
			} else {
				t.fs = fs
			}
		}

		if t.UseScpFilesystem {
			log.Info().Str("transport", "ssh").Msg("ssh uses scp (beta) instead of sftp for file transfer")
			t.fs = scp.NewFs(t.SSHClient)
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

func (t *SSHTransport) Capabilities() []capabilities.Capability {
	return []capabilities.Capability{
		capabilities.RunCommand,
		capabilities.File,
	}
}

package ssh

import (
	"net"
	"os"

	"github.com/pkg/errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/motor/motoros/ssh/scp"
	"go.mondoo.io/mondoo/motor/motoros/ssh/sftp"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"go.mondoo.io/mondoo/nexus/assets"
	"golang.org/x/crypto/ssh"

	rawsftp "github.com/pkg/sftp"
)

func New(endpoint *types.Endpoint) (*SSHTransport, error) {
	endpoint = ReadSSHConfig(endpoint)

	// ensure all required configs are set
	err := VerifyConfig(endpoint)
	if err != nil {
		return nil, err
	}

	// set default config if required
	endpoint = DefaultConfig(endpoint)

	// load known hosts and track the fingerprint of the ssh server for later identification
	knownHostsCallback, err := KnownHostsCallback()
	if err != nil {
		return nil, errors.Wrap(err, "could not read hostkey file")
	}

	var hostkey ssh.PublicKey
	hostkeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// store the hostkey for later identification
		hostkey = key

		// ignore hostkey check if the user provided an insecure flag
		if endpoint.Insecure {
			return nil
		}

		return knownHostsCallback(hostname, remote, key)
	}

	// establish connection
	conn, err := sshClientConnection(endpoint, hostkeyCallback)
	if err != nil {
		log.Debug().Err(err).Str("transport", "ssh").Str("host", endpoint.Host).Str("port", endpoint.Port).Str("user", endpoint.User).Msg("could not establish ssh session")
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

func (t *SSHTransport) FileInfo(path string) (types.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return types.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)

	if t.UseScpFilesystem {
		// scp does not preserve uid and gid
	} else {
		if stat, ok := stat.Sys().(*rawsftp.FileStat); ok {
			uid = int64(stat.UID)
			gid = int64(stat.GID)
		}
	}
	mode := stat.Mode()

	return types.FileInfoDetails{
		Mode: types.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *SSHTransport) Close() {
	if t.SSHClient != nil {
		t.SSHClient.Close()
	}
}

func (t *SSHTransport) Capabilities() capabilities.Capabilities {
	return capabilities.Capabilities{
		capabilities.RunCommand,
		capabilities.File,
	}
}

func (t *SSHTransport) Kind() assets.Kind {
	return assets.Kind_KIND_BARE_METAL
}

func (t *SSHTransport) Runtime() string {
	return ""
}

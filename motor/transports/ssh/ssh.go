package ssh

import (
	"net"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/cockroachdb/errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/cmd"
	"go.mondoo.io/mondoo/motor/transports/ssh/cat"
	"go.mondoo.io/mondoo/motor/transports/ssh/scp"
	"go.mondoo.io/mondoo/motor/transports/ssh/sftp"
	"golang.org/x/crypto/ssh"

	rawsftp "github.com/pkg/sftp"
)

func New(endpoint *transports.TransportConfig) (*SSHTransport, error) {
	endpoint = ReadSSHConfig(endpoint)

	// ensure all required configs are set
	err := VerifyConfig(endpoint)
	if err != nil {
		return nil, err
	}

	// set default config if required
	endpoint = DefaultConfig(endpoint)

	activateScp := false
	if os.Getenv("MONDOO_SSH_SCP") == "on" {
		activateScp = true
	}

	var s cmd.Wrapper
	if endpoint.Sudo != nil && endpoint.Sudo.Active {
		log.Debug().Msg("activated sudo for ssh connection")
		s = cmd.NewSudo()
	}

	if endpoint.Insecure {
		log.Debug().Msg("user allowed insecure ssh connection")
	}

	t := &SSHTransport{
		ConnectionConfig: endpoint,
		UseScpFilesystem: activateScp,
		Sudo:             s,
		kind:             endpoint.Kind,
		runtime:          endpoint.Runtime,
	}
	err = t.Connect()
	return t, err
}

// TODO: only run when ssh-agent credential is there
func DefaultConfig(cc *transports.TransportConfig) *transports.TransportConfig {
	p, err := cc.IntPort()
	// use default port if port is 0
	if err == nil && p <= 0 {
		cc.Port = "22"
	}

	// ssh config overwrite like: IdentityFile ~/.foo/identity is done in ReadSSHConfig()
	// fallback to default paths 	~/.ssh/id_rsa and ~/.ssh/id_dsa if they exist
	home, err := homedir.Dir()
	if err == nil {
		files := []string{
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_dsa"),
			// specific handling for google compute engine, see https://cloud.google.com/compute/docs/instances/connecting-to-instance
			// filepath.Join(home, ".ssh", "google_compute_engine"),
		}

		// filter keys by existence
		for i := range files {
			f := files[i]
			_, err := os.Stat(f)
			if err == nil {
				// apply the option manually
				// TODO: change username
				credential, _ := transports.NewPrivateKeyCredentialFromPath("changem", f, nil)
				cc.AddCredential(credential)
			}
		}
	}

	return cc
}

type SSHTransport struct {
	ConnectionConfig *transports.TransportConfig
	SSHClient        *ssh.Client
	fs               afero.Fs
	UseScpFilesystem bool
	HostKey          ssh.PublicKey
	Sudo             cmd.Wrapper
	kind             transports.Kind
	runtime          string
	serverVersion    string
}

func (t *SSHTransport) Connect() error {
	cc := t.ConnectionConfig

	// load known hosts and track the fingerprint of the ssh server for later identification
	knownHostsCallback, err := KnownHostsCallback()
	if err != nil {
		return errors.Wrap(err, "could not read hostkey file")
	}

	var hostkey ssh.PublicKey
	hostkeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// store the hostkey for later identification
		hostkey = key

		// ignore hostkey check if the user provided an insecure flag
		if cc.Insecure {
			return nil
		}

		return knownHostsCallback(hostname, remote, key)
	}

	// establish connection
	conn, err := sshClientConnection(cc, hostkeyCallback)
	if err != nil {
		log.Debug().Err(err).Str("transport", "ssh").Str("host", cc.Host).Str("port", cc.Port).Bool("insecure", cc.Insecure).Msg("could not establish ssh session")
		return err
	}
	t.SSHClient = conn
	t.HostKey = hostkey
	t.serverVersion = string(conn.ServerVersion())
	log.Debug().Str("transport", "ssh").Str("host", cc.Host).Str("port", cc.Port).Str("server", t.serverVersion).Msg("ssh session established")
	return nil
}

func (t *SSHTransport) Reconnect() error {
	// ensure the connections is going to be closed
	if t.SSHClient != nil {
		t.SSHClient.Close()
	}
	return t.Connect()
}

func (t *SSHTransport) RunCommand(command string) (*transports.Command, error) {
	if t.Sudo != nil {
		command = t.Sudo.Build(command)
	}

	log.Debug().Str("command", command).Str("transport", "ssh").Msg("run command")
	c := &Command{SSHTransport: t}
	return c.Exec(command)
}

func (t *SSHTransport) FS() afero.Fs {
	// if we cached an instance already, return it
	if t.fs != nil {
		return t.fs
	}

	// log the used ssh filesystem backend
	defer func() {
		log.Debug().Str("file-transfer", t.fs.Name()).Msg("initialized ssh filesystem")
	}()

	//// detect cisco network gear, they returns something like SSH-2.0-Cisco-1.25
	//// NOTE: we need to understand why this happens
	//if strings.Contains(strings.ToLower(t.serverVersion), "cisco") {
	//	log.Debug().Msg("detected cisco device, deactivate file system support")
	//	t.fs = &fsutil.NoFs{}
	//	return t.fs
	//}

	// if any priviledge elevation is used, we have no other chance as to use command-based file transfer
	if t.Sudo != nil {
		t.fs = cat.New(t)
		return t.fs
	}

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
			return t.fs
		}
	}

	if t.UseScpFilesystem {
		t.fs = scp.NewFs(t, t.SSHClient)
		return t.fs
	}

	// always fallback to catfs, slow but it works
	t.fs = cat.New(t)
	return t.fs
}

func (t *SSHTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)

	if t.Sudo != nil || t.UseScpFilesystem {
		if stat, ok := stat.Sys().(*transports.FileInfo); ok {
			uid = int64(stat.Uid)
			gid = int64(stat.Gid)
		}
	} else {
		if stat, ok := stat.Sys().(*rawsftp.FileStat); ok {
			uid = int64(stat.UID)
			gid = int64(stat.GID)
		}
	}
	mode := stat.Mode()

	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
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

func (t *SSHTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_RunCommand,
		transports.Capability_File,
	}
}

func (t *SSHTransport) Kind() transports.Kind {
	return t.kind
}

func (t *SSHTransport) Runtime() string {
	return t.runtime
}

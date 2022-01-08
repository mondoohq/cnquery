package ssh

import (
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	rawsftp "github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/cmd"
	"go.mondoo.io/mondoo/motor/transports/ssh/cat"
	"go.mondoo.io/mondoo/motor/transports/ssh/scp"
	"go.mondoo.io/mondoo/motor/transports/ssh/sftp"
	"golang.org/x/crypto/ssh"
)

var _ transports.Transport = (*SSHTransport)(nil)

func New(tc *transports.TransportConfig) (*SSHTransport, error) {
	tc = ReadSSHConfig(tc)

	// ensure all required configs are set
	err := VerifyConfig(tc)
	if err != nil {
		return nil, err
	}

	activateScp := false
	if os.Getenv("MONDOO_SSH_SCP") == "on" {
		activateScp = true
	}

	if tc.Insecure {
		log.Debug().Msg("user allowed insecure ssh connection")
	}

	t := &SSHTransport{
		ConnectionConfig: tc,
		UseScpFilesystem: activateScp,
		kind:             tc.Kind,
		runtime:          tc.Runtime,
	}
	err = t.Connect()
	if err != nil {
		return nil, err
	}

	var s cmd.Wrapper
	// check uid of user and disable sudo if uid is 0
	if tc.Sudo != nil && tc.Sudo.Active {
		// the id command may not be available, eg. if ssh is used with windows
		out, _ := t.RunCommand("id -u")
		stdout, _ := ioutil.ReadAll(out.Stdout)
		// just check for the explicit positive case, otherwise just activate sudo
		// we check sudo in VerifyConnection
		if string(stdout) != "0" {
			// configure sudo
			log.Debug().Msg("activated sudo for ssh connection")
			s = cmd.NewSudo()
		}
	}
	t.Sudo = s

	// verify connection
	vErr := t.VerifyConnection()
	// NOTE: for now we do not enforce connection verification to ensure we cover edge-cases
	// TODO: in following minor version bumps, we want to enforce this behavior to ensure proper scans
	if vErr != nil {
		log.Warn().Err(vErr).Send()
	}

	return t, nil
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

	// we always want to ensure we use the default port if nothing was specified
	if cc.Port == 0 {
		cc.Port = 22
	}

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
	conn, _, err := establishClientConnection(cc, hostkeyCallback)
	if err != nil {
		log.Debug().Err(err).Str("transport", "ssh").Str("host", cc.Host).Int32("port", cc.Port).Bool("insecure", cc.Insecure).Msg("could not establish ssh session")
		return err
	}
	t.SSHClient = conn
	t.HostKey = hostkey
	t.serverVersion = string(conn.ServerVersion())
	log.Debug().Str("transport", "ssh").Str("host", cc.Host).Int32("port", cc.Port).Str("server", t.serverVersion).Msg("ssh session established")
	return nil
}

func (t *SSHTransport) VerifyConnection() error {
	var out *transports.Command
	var err error

	if t.Sudo != nil {
		// Wrap sudo command, to see proper error messages. We set /dev/null to disable stdin
		command := "sh -c '" + t.Sudo.Build("echo 'hi'") + " < /dev/null'"
		out, err = t.runRawCommand(command)
	} else {
		out, err = t.runRawCommand("echo 'hi'")
		if err != nil {
			return err
		}
	}

	if out.ExitStatus == 0 {
		return nil
	}

	stderr, _ := ioutil.ReadAll(out.Stderr)
	errMsg := string(stderr)

	// sample messages are:
	// sudo: a terminal is required to read the password; either use the -S option to read from standard input or configure an askpass helper
	// sudo: a password is required
	switch {
	case strings.Contains(errMsg, "not found"):
		return errors.New("sudo command is missing on target")
	case strings.Contains(errMsg, "a password is required"):
		return errors.New("could not establish connection: sudo password is not supported yet, configure password-less sudo")
	default:
		return errors.New("could not establish connection: " + errMsg)
	}
}

// Reconnect closes a possible current connection and re-establishes a new connection
func (t *SSHTransport) Reconnect() error {
	t.Close()
	return t.Connect()
}

func (t *SSHTransport) runRawCommand(command string) (*transports.Command, error) {
	log.Debug().Str("command", command).Str("transport", "ssh").Msg("run command")
	c := &Command{SSHTransport: t}
	return c.Exec(command)
}

func (t *SSHTransport) RunCommand(command string) (*transports.Command, error) {
	if t.Sudo != nil {
		command = t.Sudo.Build(command)
	}
	return t.runRawCommand(command)
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
		fs, err := sftp.New(t, t.SSHClient)
		if err != nil {
			log.Info().Msg("use scp instead of sftp")
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

func (t *SSHTransport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.HostnameDetector,
		transports.SSHHostKeyDetector,
		transports.CloudDetector,
	}
}

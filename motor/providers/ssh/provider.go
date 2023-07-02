package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"errors"
	rawsftp "github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/cmd"
	"go.mondoo.com/cnquery/motor/providers/ssh/cat"
	"go.mondoo.com/cnquery/motor/providers/ssh/scp"
	"go.mondoo.com/cnquery/motor/providers/ssh/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	_ providers.Instance                  = (*Provider)(nil)
	_ providers.PlatformIdentifier        = (*Provider)(nil)
	_ os_provider.OperatingSystemProvider = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	host := pCfg.GetHost()
	// ipv6 addresses w/o the surrounding [] will eventually error
	// so check whether we have an ipv6 address by parsing it (the
	// parsing will fail if the string DOES have the []s) and adding
	// the []s
	ip := net.ParseIP(host)
	if ip != nil && ip.To4() == nil {
		pCfg.Host = fmt.Sprintf("[%s]", host)
	}

	pCfg = ReadSSHConfig(pCfg)

	// ensure all required configs are set
	err := VerifyConfig(pCfg)
	if err != nil {
		return nil, err
	}

	activateScp := false
	if os.Getenv("MONDOO_SSH_SCP") == "on" || pCfg.Options["ssh_scp"] == "on" {
		activateScp = true
	}

	if pCfg.Insecure {
		log.Debug().Msg("user allowed insecure ssh connection")
	}

	t := &Provider{
		ConnectionConfig: pCfg,
		UseScpFilesystem: activateScp,
		kind:             pCfg.Kind,
		runtime:          pCfg.Runtime,
	}
	err = t.Connect()
	if err != nil {
		return nil, err
	}

	var s cmd.Wrapper
	// check uid of user and disable sudo if uid is 0
	if pCfg.Sudo != nil && pCfg.Sudo.Active {
		// the id command may not be available, eg. if ssh is used with windows
		out, _ := t.RunCommand("id -u")
		stdout, _ := io.ReadAll(out.Stdout)
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

type Provider struct {
	ConnectionConfig *providers.Config
	SSHClient        *ssh.Client
	fs               afero.Fs
	UseScpFilesystem bool
	HostKey          ssh.PublicKey
	Sudo             cmd.Wrapper
	kind             providers.Kind
	runtime          string
	serverVersion    string
}

func (p *Provider) Connect() error {
	cc := p.ConnectionConfig

	// we always want to ensure we use the default port if nothing was specified
	if cc.Port == 0 {
		cc.Port = 22
	}

	// load known hosts and track the fingerprint of the ssh server for later identification
	knownHostsCallback, err := KnownHostsCallback()
	if err != nil {
		return errors.Join(err, errors.New("could not read hostkey file"))
	}

	var hostkey ssh.PublicKey
	hostkeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// store the hostkey for later identification
		hostkey = key

		// ignore hostkey check if the user provided an insecure flag
		if cc.Insecure {
			return nil
		}

		// knownhost.New returns a ssh.CertChecker which does not work with all ssh.HostKey types
		// especially the newer edcsa keys (ssh.curve25519sha256) are not well supported.
		// https://github.com/golang/crypto/blob/master/ssh/knownhosts/knownhosts.go#L417-L436
		// creates the CertChecker which requires an instance of Certificate
		// https://github.com/golang/crypto/blob/master/ssh/certs.go#L326-L348
		// https://github.com/golang/crypto/blob/master/ssh/keys.go#L271-L283
		// therefore it is best to skip the checking for now since it forces users to set the insecure flag otherwise
		// TODO: implement custom host-key checking for normal public keys as well
		_, ok := key.(*ssh.Certificate)
		if !ok {
			log.Debug().Msg("skip hostkey check the hostkey since the algo is not supported yet")
			return nil
		}

		err := knownHostsCallback(hostname, remote, key)
		if err != nil {
			log.Debug().Err(err).Str("hostname", hostname).Str("ip", remote.String()).Msg("check known host")
		}
		return err
	}

	// establish connection
	conn, _, err := establishClientConnection(cc, hostkeyCallback)
	if err != nil {
		log.Debug().Err(err).Str("provider", "ssh").Str("host", cc.Host).Int32("port", cc.Port).Bool("insecure", cc.Insecure).Msg("could not establish ssh session")
		if strings.ContainsAny(cc.Host, "[]") {
			log.Info().Str("host", cc.Host).Int32("port", cc.Port).Msg("ensure proper []s when combining IPv6 with port numbers")
		}
		return err
	}
	p.SSHClient = conn
	p.HostKey = hostkey
	p.serverVersion = string(conn.ServerVersion())
	log.Debug().Str("provider", "ssh").Str("host", cc.Host).Int32("port", cc.Port).Str("server", p.serverVersion).Msg("ssh session established")
	return nil
}

func (p *Provider) VerifyConnection() error {
	var out *os_provider.Command
	var err error

	if p.Sudo != nil {
		// Wrap sudo command, to see proper error messages. We set /dev/null to disable stdin
		command := "sh -c '" + p.Sudo.Build("echo 'hi'") + " < /dev/null'"
		out, err = p.runRawCommand(command)
	} else {
		out, err = p.runRawCommand("echo 'hi'")
		if err != nil {
			return err
		}
	}

	if out.ExitStatus == 0 {
		return nil
	}

	stderr, _ := io.ReadAll(out.Stderr)
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
func (p *Provider) Reconnect() error {
	p.Close()
	return p.Connect()
}

func (p *Provider) runRawCommand(command string) (*os_provider.Command, error) {
	log.Debug().Str("command", command).Str("provider", "ssh").Msg("run command")
	c := &Command{SSHProvider: p}
	return c.Exec(command)
}

func (p *Provider) RunCommand(command string) (*os_provider.Command, error) {
	if p.Sudo != nil {
		command = p.Sudo.Build(command)
	}
	return p.runRawCommand(command)
}

func (p *Provider) FS() afero.Fs {
	// if we cached an instance already, return it
	if p.fs != nil {
		return p.fs
	}

	// log the used ssh filesystem backend
	defer func() {
		log.Debug().Str("file-transfer", p.fs.Name()).Msg("initialized ssh filesystem")
	}()

	//// detect cisco network gear, they returns something like SSH-2.0-Cisco-1.25
	//// NOTE: we need to understand why this happens
	//if strings.Contains(strings.ToLower(t.serverVersion), "cisco") {
	//	log.Debug().Msg("detected cisco device, deactivate file system support")
	//	t.fs = &fsutil.NoFs{}
	//	return t.fs
	//}

	// if any privilege elevation is used, we have no other chance as to use command-based file transfer
	if p.Sudo != nil {
		p.fs = cat.New(p)
		return p.fs
	}

	// we always try to use sftp first (if scp is not user-enforced)
	// and we also fallback to scp if sftp does not work
	if !p.UseScpFilesystem {
		fs, err := sftp.New(p, p.SSHClient)
		if err != nil {
			log.Info().Msg("use scp instead of sftp")
			// enable fallback
			p.UseScpFilesystem = true
		} else {
			p.fs = fs
			return p.fs
		}
	}

	if p.UseScpFilesystem {
		p.fs = scp.NewFs(p, p.SSHClient)
		return p.fs
	}

	// always fallback to catfs, slow but it works
	p.fs = cat.New(p)
	return p.fs
}

func (p *Provider) FileInfo(path string) (os_provider.FileInfoDetails, error) {
	fs := p.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return os_provider.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)

	if p.Sudo != nil || p.UseScpFilesystem {
		if stat, ok := stat.Sys().(*os_provider.FileInfo); ok {
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

	return os_provider.FileInfoDetails{
		Mode: os_provider.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (p *Provider) Close() {
	if p.SSHClient != nil {
		p.SSHClient.Close()
	}
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_RunCommand,
		providers.Capability_File,
	}
}

func (p *Provider) Kind() providers.Kind {
	return p.kind
}

func (p *Provider) Runtime() string {
	return p.runtime
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
		providers.HostnameDetector,
		providers.CloudDetector,
	}
}

func (p *Provider) Identifier() (string, error) {
	return PlatformIdentifier(p.HostKey), nil
}

func PlatformIdentifier(publicKey ssh.PublicKey) string {
	fingerprint := ssh.FingerprintSHA256(publicKey)
	fingerprint = strings.Replace(fingerprint, ":", "-", 1)
	identifier := "//platformid.api.mondoo.app/runtime/ssh/hostkey/" + fingerprint
	return identifier
}

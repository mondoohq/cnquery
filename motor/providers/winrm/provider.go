package winrm

import (
	"bytes"
	"errors"
	"os"
	"time"

	"github.com/masterzen/winrm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	os_provider "go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/motor/providers/winrm/cat"
	"go.mondoo.io/mondoo/motor/vault"
)

var _ providers.Instance = (*Provider)(nil)

func VerifyConfig(pCfg *providers.Config) (*winrm.Endpoint, error) {
	if pCfg.Backend != providers.ProviderType_WINRM {
		return nil, errors.New("only winrm backend for winrm transport supported")
	}

	winrmEndpoint := &winrm.Endpoint{
		Host:     pCfg.Host,
		Port:     int(pCfg.Port),
		Insecure: pCfg.Insecure,
		HTTPS:    true,
		Timeout:  time.Duration(0),
	}

	return winrmEndpoint, nil
}

func DefaultConfig(endpoint *winrm.Endpoint) *winrm.Endpoint {
	// use default port if port is 0
	if endpoint.Port <= 0 {
		endpoint.Port = 5986
	}

	if endpoint.Port == 5985 {
		log.Warn().Msg("winrm port 5985 is using http communication instead of https, passwords are not encrypted")
		endpoint.HTTPS = false
	}

	if os.Getenv("WINRM_DISABLE_HTTPS") == "true" {
		log.Warn().Msg("WINRM_DISABLE_HTTPS is set, winrm is using http communication instead of https, passwords are not encrypted")
		endpoint.HTTPS = false
	}

	return endpoint
}

// New creates a winrm client and establishes a connection to verify the connection
func New(pCfg *providers.Config) (*Provider, error) {
	// ensure all required configs are set
	winrmEndpoint, err := VerifyConfig(pCfg)
	if err != nil {
		return nil, err
	}

	// set default config if required
	winrmEndpoint = DefaultConfig(winrmEndpoint)

	params := winrm.DefaultParameters
	params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }

	// search for password secret
	c, err := vault.GetPassword(pCfg.Credentials)
	if err != nil {
		return nil, errors.New("missing password for winrm transport")
	}

	client, err := winrm.NewClientWithParameters(winrmEndpoint, c.User, string(c.Secret), params)
	if err != nil {
		return nil, err
	}

	// test connection
	log.Debug().Str("user", c.User).Str("host", pCfg.Host).Msg("winrm> connecting to remote shell via WinRM")
	shell, err := client.CreateShell()
	if err != nil {
		return nil, err
	}

	err = shell.Close()
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("winrm> connection established")
	return &Provider{
		Endpoint: winrmEndpoint,
		Client:   client,
		kind:     pCfg.Kind,
		runtime:  pCfg.Runtime,
	}, nil
}

type Provider struct {
	Endpoint *winrm.Endpoint
	Client   *winrm.Client
	kind     providers.Kind
	runtime  string
	fs       afero.Fs
}

func (p *Provider) RunCommand(command string) (*os_provider.Command, error) {
	log.Debug().Str("command", command).Str("provider", "winrm").Msg("winrm> run command")

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	mcmd := &os_provider.Command{
		Command: command,
		Stdout:  stdoutBuffer,
		Stderr:  stderrBuffer,
	}

	// Note: winrm does not return err of the command was executed with a non-zero exit code
	exitCode, err := p.Client.Run(command, stdoutBuffer, stderrBuffer)
	if err != nil {
		log.Error().Err(err).Str("command", command).Msg("could not execute winrm command")
		return mcmd, err
	}

	mcmd.ExitStatus = exitCode
	return mcmd, nil
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
	mode := stat.Mode()

	return os_provider.FileInfoDetails{
		Mode: os_provider.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (p *Provider) FS() afero.Fs {
	if p.fs == nil {
		p.fs = cat.New(p)
	}
	return p.fs
}

func (p *Provider) Close() {
	// nothing to do yet
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
		providers.HostnameDetector,
		providers.CloudDetector,
	}
}

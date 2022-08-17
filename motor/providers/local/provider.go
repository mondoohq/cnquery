package local

import (
	"io"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/os/cmd"
	"go.mondoo.io/mondoo/motor/providers/ssh/cat"
)

var _ providers.Transport = (*Provider)(nil)

func New() (*Provider, error) {
	return NewWithConfig(&providers.Config{})
}

func NewWithConfig(pCfg *providers.Config) (*Provider, error) {
	// expect unix shell by default
	shell := []string{"sh", "-c"}

	if runtime.GOOS == "windows" {
		// It does not make any sense to use cmd as default shell
		// shell = []string{"cmd", "/C"}
		shell = []string{"powershell", "-c"}
	}

	p := &Provider{
		shell: shell,
		// kind:    endpoint.Kind,
		// runtime: endpoint.Runtime,
	}

	var s cmd.Wrapper
	if pCfg != nil && pCfg.Sudo != nil && pCfg.Sudo.Active {
		// the id command may not be available, eg. if ssh is used with windows
		out, _ := p.RunCommand("id -u")
		stdout, _ := io.ReadAll(out.Stdout)
		// just check for the explicit positive case, otherwise just activate sudo
		// we check sudo in VerifyConnection
		if string(stdout) != "0" {
			// configure sudo
			log.Debug().Msg("activated sudo for local connection")
			s = cmd.NewSudo()
		}
	}
	p.Sudo = s

	return p, nil
}

type Provider struct {
	shell   []string
	fs      afero.Fs
	Sudo    cmd.Wrapper
	kind    providers.Kind
	runtime string
}

func (p *Provider) RunCommand(command string) (*providers.Command, error) {
	log.Debug().Msgf("local> run command %s", command)
	if p.Sudo != nil {
		command = p.Sudo.Build(command)
	}
	c := &cmd.Command{Shell: p.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (p *Provider) FS() afero.Fs {
	if p.fs != nil {
		return p.fs
	}

	if p.Sudo != nil {
		p.fs = cat.New(p)
		return p.fs
	}

	p.fs = afero.NewOsFs()
	return p.fs
}

func (p *Provider) FileInfo(path string) (providers.FileInfoDetails, error) {
	fs := p.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return providers.FileInfoDetails{}, err
	}

	uid, gid := p.fileowner(stat)

	mode := stat.Mode()
	return providers.FileInfoDetails{
		Mode: providers.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (p *Provider) Close() {
	// TODO: we need to close all commands and file handles
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

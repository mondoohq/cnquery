package fs

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
)

var _ providers.Transport = (*Provider)(nil)

func NewWithClose(endpoint *providers.Config, closeFN func()) (*Provider, error) {
	mountDir := endpoint.Host + endpoint.Path
	log.Info().Str("mountdir", mountDir).Msg("load fs")

	return &Provider{
		MountedDir:   mountDir,
		closeFN:      closeFN,
		tcPlatformId: endpoint.PlatformId,
		fs:           NewMountedFs(mountDir),
	}, nil
}

func New(endpoint *providers.Config) (*Provider, error) {
	mountDir := endpoint.Host + endpoint.Path
	log.Info().Str("mountdir", mountDir).Msg("load fs")

	return &Provider{
		MountedDir:   mountDir,
		tcPlatformId: endpoint.PlatformId,
		fs:           NewMountedFs(mountDir),
	}, nil
}

type Provider struct {
	MountedDir   string
	fs           afero.Fs
	kind         providers.Kind
	runtime      string
	tcPlatformId string
	closeFN      func()
}

func (p *Provider) RunCommand(command string) (*providers.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (p *Provider) FS() afero.Fs {
	if p.fs == nil {
		p.fs = NewMountedFs(p.MountedDir)
	}
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
	if p.closeFN != nil {
		p.closeFN()
	}
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_FileSearch,
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
	if p.tcPlatformId != "" {
		return []providers.PlatformIdDetector{
			providers.TransportPlatformIdentifierDetector,
		}
	}
	return []providers.PlatformIdDetector{
		providers.HostnameDetector,
	}
}

func (p *Provider) Identifier() (string, error) {
	if p.tcPlatformId == "" {
		return "", errors.New("not platform id provided")
	}
	return p.tcPlatformId, nil
}

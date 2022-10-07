package fs

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/os"
)

var (
	_ providers.Instance         = (*Provider)(nil)
	_ os.OperatingSystemProvider = (*Provider)(nil)
)

func NewWithClose(endpoint *providers.Config, closeFN func()) (*Provider, error) {
	mountDir := endpoint.Host + endpoint.Path
	log.Info().Str("mountdir", mountDir).Msg("load fs")

	return &Provider{
		MountedDir:   mountDir,
		closeFN:      closeFN,
		tcPlatformId: endpoint.PlatformId,
		fs:           NewMountedFs(mountDir),
		runtime:      endpoint.Runtime,
	}, nil
}

func New(endpoint *providers.Config) (*Provider, error) {
	return NewWithClose(endpoint, nil)
}

type Provider struct {
	MountedDir   string
	fs           afero.Fs
	runtime      string
	kind         providers.Kind
	tcPlatformId string
	closeFN      func()
}

func (p *Provider) RunCommand(command string) (*os.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (p *Provider) FS() afero.Fs {
	if p.fs == nil {
		p.fs = NewMountedFs(p.MountedDir)
	}
	return p.fs
}

func (p *Provider) FileInfo(path string) (os.FileInfoDetails, error) {
	fs := p.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return os.FileInfoDetails{}, err
	}

	uid, gid := p.fileowner(stat)

	mode := stat.Mode()
	return os.FileInfoDetails{
		Mode: os.FileModeDetails{mode},
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

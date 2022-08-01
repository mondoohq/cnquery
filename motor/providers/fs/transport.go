package fs

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
)

var _ providers.Transport = (*FsTransport)(nil)

func NewWithClose(endpoint *providers.TransportConfig, closeFN func()) (*FsTransport, error) {
	mountDir := endpoint.Host + endpoint.Path
	log.Info().Str("mountdir", mountDir).Msg("load fs")

	return &FsTransport{
		MountedDir:   mountDir,
		closeFN:      closeFN,
		tcPlatformId: endpoint.PlatformId,
		fs:           NewMountedFs(mountDir),
	}, nil
}

func New(endpoint *providers.TransportConfig) (*FsTransport, error) {
	mountDir := endpoint.Host + endpoint.Path
	log.Info().Str("mountdir", mountDir).Msg("load fs")

	return &FsTransport{
		MountedDir:   mountDir,
		tcPlatformId: endpoint.PlatformId,
		fs:           NewMountedFs(mountDir),
	}, nil
}

type FsTransport struct {
	MountedDir   string
	fs           afero.Fs
	kind         providers.Kind
	runtime      string
	tcPlatformId string
	closeFN      func()
}

func (t *FsTransport) RunCommand(command string) (*providers.Command, error) {
	return nil, errors.New("filesearch transport does not implement RunCommand")
}

func (t *FsTransport) FS() afero.Fs {
	if t.fs == nil {
		t.fs = NewMountedFs(t.MountedDir)
	}
	return t.fs
}

func (t *FsTransport) FileInfo(path string) (providers.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return providers.FileInfoDetails{}, err
	}

	uid, gid := t.fileowner(stat)

	mode := stat.Mode()
	return providers.FileInfoDetails{
		Mode: providers.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *FsTransport) Close() {
	if t.closeFN != nil {
		t.closeFN()
	}
}

func (t *FsTransport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_FileSearch,
		providers.Capability_File,
	}
}

func (t *FsTransport) Kind() providers.Kind {
	return t.kind
}

func (t *FsTransport) Runtime() string {
	return t.runtime
}

func (t *FsTransport) PlatformIdDetectors() []providers.PlatformIdDetector {
	if t.tcPlatformId != "" {
		return []providers.PlatformIdDetector{
			providers.TransportPlatformIdentifierDetector,
		}
	}
	return []providers.PlatformIdDetector{
		providers.HostnameDetector,
	}
}

func (t *FsTransport) Identifier() (string, error) {
	if t.tcPlatformId == "" {
		return "", errors.New("not platform id provided")
	}
	return t.tcPlatformId, nil
}

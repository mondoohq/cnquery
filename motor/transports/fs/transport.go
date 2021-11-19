package fs

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
)

var _ transports.Transport = (*FsTransport)(nil)

func NewWithClose(endpoint *transports.TransportConfig, closeFN func()) (*FsTransport, error) {
	log.Info().Str("mountdir", endpoint.Host+endpoint.Path).Msg("load fs")

	return &FsTransport{
		MountedDir:   endpoint.Host + endpoint.Path,
		closeFN:      closeFN,
		tcPlatformId: endpoint.PlatformId,
	}, nil
}

func New(endpoint *transports.TransportConfig) (*FsTransport, error) {
	log.Info().Str("mountdir", endpoint.Host+endpoint.Path).Msg("load fs")

	return &FsTransport{
		MountedDir:   endpoint.Host + endpoint.Path,
		tcPlatformId: endpoint.PlatformId,
	}, nil
}

type FsTransport struct {
	MountedDir   string
	fs           afero.Fs
	kind         transports.Kind
	runtime      string
	tcPlatformId string
	closeFN      func()
}

func (t *FsTransport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("filesearch transport does not implement RunCommand")
}

func (t *FsTransport) FS() afero.Fs {
	if t.fs == nil {
		t.fs = NewMountedFs(t.MountedDir)
	}
	return t.fs
}

func (t *FsTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid, gid := t.fileowner(stat)

	mode := stat.Mode()
	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
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

func (t *FsTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_FileSearch,
		transports.Capability_File,
	}
}

func (t *FsTransport) Kind() transports.Kind {
	return t.kind
}

func (t *FsTransport) Runtime() string {
	return t.runtime
}

func (t *FsTransport) PlatformIdDetectors() []transports.PlatformIdDetector {
	if t.tcPlatformId != "" {
		return []transports.PlatformIdDetector{
			transports.TransportPlatformIdentifierDetector,
		}
	}
	return []transports.PlatformIdDetector{
		transports.HostnameDetector,
	}
}

func (t *FsTransport) Identifier() (string, error) {
	if t.tcPlatformId == "" {
		return "", errors.New("not platform id provided")
	}
	return t.tcPlatformId, nil
}

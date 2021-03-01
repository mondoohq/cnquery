package fs

import (
	"bytes"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
)

func New(endpoint *transports.TransportConfig) (*FsTransport, error) {
	log.Info().Str("mountdir", endpoint.Host+endpoint.Path).Msg("load fs")

	return &FsTransport{
		mountedDir: endpoint.Host + endpoint.Path,
	}, nil
}

type FsTransport struct {
	mountedDir string
	fs         afero.Fs
	kind       transports.Kind
	runtime    string
}

func (t *FsTransport) RunCommand(command string) (*transports.Command, error) {
	// TODO: switch to error state
	res := transports.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (t *FsTransport) FS() afero.Fs {
	if t.fs == nil {
		t.fs = NewMountedFs(t.mountedDir)
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

func (t *FsTransport) Close() {}

func (t *FsTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_File,
	}
}

func (t *FsTransport) Kind() transports.Kind {
	return t.kind
}

func (t *FsTransport) Runtime() string {
	return t.runtime
}

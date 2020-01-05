package tar

import (
	"archive/tar"
	"bytes"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"go.mondoo.io/mondoo/nexus/assets"

	"io"
	"os"
)

func New(endpoint *types.Endpoint) (*Transport, error) {
	return NewWithClose(endpoint, nil)
}

func NewWithClose(endpoint *types.Endpoint, close func()) (*Transport, error) {
	t := &Transport{
		Fs:      NewFs(endpoint.Path),
		CloseFN: close,
	}

	var err error
	if endpoint != nil && len(endpoint.Path) > 0 {
		err := t.LoadFile(endpoint.Path)
		if err != nil {
			log.Error().Err(err).Str("tar", endpoint.Path).Msg("tar> could not load tar file")
			return nil, err
		}
	}
	return t, err
}

// Transport loads tar files and make them available
type Transport struct {
	Fs      *FS
	CloseFN func()
}

func (m *Transport) RunCommand(command string) (*types.Command, error) {
	res := types.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (t *Transport) FS() afero.Fs {
	return t.Fs
}

func (t *Transport) File(path string) (afero.File, error) {
	return t.FS().Open(path)
}

func (t *Transport) Close() {
	if t.CloseFN != nil {
		t.CloseFN()
	}
}

func (t *Transport) Capabilities() []capabilities.Capability {
	return []capabilities.Capability{
		capabilities.File,
	}
}

func (t *Transport) Kind() assets.Kind {
	return assets.Kind_KIND_UNKNOWN
}

func (t *Transport) Runtime() string {
	return ""
}

func (t *Transport) Load(stream io.Reader) error {
	tr := tar.NewReader(stream)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("tar> error reading tar stream")
			return err
		}

		path := Abs(h.Name)
		t.Fs.FileMap[path] = h
	}
	log.Debug().Int("files", len(t.Fs.FileMap)).Msg("tar> successfully loaded")
	return nil
}

func (t *Transport) LoadFile(path string) error {
	log.Debug().Str("path", path).Msg("tar> load tar file into backend")

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return err
	}

	return t.Load(f)
}

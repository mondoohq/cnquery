package tar

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/v1/mutate"

	"go.mondoo.io/mondoo/motor/transports/container/cache"

	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motorid/containerid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

const OPTION_FILE = "file"

var (
	_ transports.Transport                   = (*Transport)(nil)
	_ transports.TransportPlatformIdentifier = (*Transport)(nil)
)

func New(endpoint *transports.TransportConfig) (*Transport, error) {
	return NewWithClose(endpoint, nil)
}

// NewWithReader provides a transport from a container image stream
func NewWithReader(rc io.ReadCloser, close func()) (*Transport, error) {
	// we cache the flattened image locally
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	// we return a pure tar image
	filename := f.Name()

	err = cache.StreamToTmpFile(rc, f)
	if err != nil {
		os.Remove(filename)
		return nil, err
	}

	return NewWithClose(&transports.TransportConfig{
		Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
		Runtime: transports.RUNTIME_DOCKER_IMAGE,
		Options: map[string]string{
			OPTION_FILE: filename,
		},
	}, func() {
		// remove temporary file on stream close
		os.Remove(filename)
	})
}

func NewWithClose(endpoint *transports.TransportConfig, closeFn func()) (*Transport, error) {
	if endpoint == nil || len(endpoint.Options[OPTION_FILE]) == 0 {
		return nil, errors.New("endpoint cannot be empty")
	}

	filename := endpoint.Options[OPTION_FILE]
	var identifier string

	// try to determine if the tar is a container image
	img, iErr := tarball.ImageFromPath(filename, nil)
	if iErr == nil {
		hash, err := img.Digest()
		if err != nil {
			return nil, err
		}
		identifier = containerid.MondooContainerImageID(hash.String())
		// if it is a container image, we need to transform the tar first, so that all layers are flattened
		trans, err := NewWithReader(mutate.Extract(img), closeFn)
		if err != nil {
			return nil, err
		}
		trans.PlatformIdentifier = identifier
		return trans, nil
	} else {
		hash, err := fsutil.LocalFileSha256(filename)
		if err != nil {
			return nil, err
		}
		identifier = "//platformid.api.mondoo.app/runtime/tar/hash/" + hash

		t := &Transport{
			Fs:              NewFs(filename),
			CloseFN:         closeFn,
			PlatformKind:    endpoint.Kind,
			PlatformRuntime: endpoint.Runtime,
		}

		err = t.LoadFile(filename)
		if err != nil {
			log.Error().Err(err).Str("tar", filename).Msg("tar> could not load tar file")
			return nil, err
		}

		t.PlatformIdentifier = identifier
		return t, nil
	}
}

func PlatformID(filename string) (string, error) {
	var identifier string
	// try to determine if the tar is a container image
	img, iErr := tarball.ImageFromPath(filename, nil)
	if iErr == nil {
		hash, err := img.Digest()
		if err != nil {
			return "", err
		}
		identifier = containerid.MondooContainerImageID(hash.String())
	} else {
		hash, err := fsutil.LocalFileSha256(filename)
		if err != nil {
			return "", err
		}
		identifier = "//platformid.api.mondoo.app/runtime/tar/hash/" + hash
	}
	return identifier, nil
}

// Transport loads tar files and make them available
type Transport struct {
	Fs      *FS
	CloseFN func()
	// fields are exposed since the tar backend is re-used for the docker backend
	PlatformKind         transports.Kind
	PlatformRuntime      string
	PlatformIdentifier   string
	PlatformArchitecture string
	// optional metadata to store additional information
	Metadata struct {
		Name   string
		Labels map[string]string
	}
}

func (t *Transport) Identifier() (string, error) {
	return t.PlatformIdentifier, nil
}

func (t *Transport) Labels() map[string]string {
	return t.Metadata.Labels
}

func (t *Transport) PlatformName() string {
	return t.Metadata.Name
}

func (m *Transport) RunCommand(command string) (*transports.Command, error) {
	// TODO: switch to error state
	res := transports.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (t *Transport) FS() afero.Fs {
	return t.Fs
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*tar.Header); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}
	mode := stat.Mode()

	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *Transport) Close() {
	if t.CloseFN != nil {
		t.CloseFN()
	}
}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_File,
		transports.Capability_FileSearch,
	}
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
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Load(f)
}

func (t *Transport) Kind() transports.Kind {
	return t.PlatformKind
}

func (t *Transport) Runtime() string {
	return t.PlatformRuntime
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}

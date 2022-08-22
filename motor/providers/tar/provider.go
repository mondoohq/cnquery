package tar

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motorid/containerid"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/container/cache"
	os_provider "go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/motor/providers/os/fsutil"
)

const OPTION_FILE = "file"

var (
	_ providers.Transport          = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(endpoint *providers.Config) (*Provider, error) {
	return NewWithClose(endpoint, nil)
}

// NewWithReader provides a tar provider from a container image stream
func NewWithReader(rc io.ReadCloser, close func()) (*Provider, error) {
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

	return NewWithClose(&providers.Config{
		Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
		Runtime: providers.RUNTIME_DOCKER_IMAGE,
		Options: map[string]string{
			OPTION_FILE: filename,
		},
	}, func() {
		// remove temporary file on stream close
		os.Remove(filename)
	})
}

func NewWithClose(pCfg *providers.Config, closeFn func()) (*Provider, error) {
	if pCfg == nil || len(pCfg.Options[OPTION_FILE]) == 0 {
		return nil, errors.New("endpoint cannot be empty")
	}

	filename := pCfg.Options[OPTION_FILE]
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
		p, err := NewWithReader(mutate.Extract(img), closeFn)
		if err != nil {
			return nil, err
		}
		p.PlatformIdentifier = identifier
		return p, nil
	} else {
		hash, err := fsutil.LocalFileSha256(filename)
		if err != nil {
			return nil, err
		}
		identifier = "//platformid.api.mondoo.app/runtime/tar/hash/" + hash

		p := &Provider{
			Fs:              NewFs(filename),
			CloseFN:         closeFn,
			PlatformKind:    pCfg.Kind,
			PlatformRuntime: pCfg.Runtime,
		}

		err = p.LoadFile(filename)
		if err != nil {
			log.Error().Err(err).Str("tar", filename).Msg("tar> could not load tar file")
			return nil, err
		}

		p.PlatformIdentifier = identifier
		return p, nil
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

// Provider loads tar files and make them available
type Provider struct {
	Fs      *FS
	CloseFN func()
	// fields are exposed since the tar backend is re-used for the docker backend
	PlatformKind         providers.Kind
	PlatformRuntime      string
	PlatformIdentifier   string
	PlatformArchitecture string
	// optional metadata to store additional information
	Metadata struct {
		Name   string
		Labels map[string]string
	}
}

func (p *Provider) Identifier() (string, error) {
	return p.PlatformIdentifier, nil
}

func (p *Provider) Labels() map[string]string {
	return p.Metadata.Labels
}

func (t *Provider) PlatformName() string {
	return t.Metadata.Name
}

func (p *Provider) RunCommand(command string) (*os_provider.Command, error) {
	// TODO: switch to error state
	res := os_provider.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (p *Provider) FS() afero.Fs {
	return p.Fs
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
	if stat, ok := stat.Sys().(*tar.Header); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
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
	if p.CloseFN != nil {
		p.CloseFN()
	}
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_File,
		providers.Capability_FileSearch,
	}
}

func (p *Provider) Load(stream io.Reader) error {
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
		p.Fs.FileMap[path] = h
	}
	log.Debug().Int("files", len(p.Fs.FileMap)).Msg("tar> successfully loaded")
	return nil
}

func (p *Provider) LoadFile(path string) error {
	log.Debug().Str("path", path).Msg("tar> load tar file into backend")

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.Load(f)
}

func (p *Provider) Kind() providers.Kind {
	return p.PlatformKind
}

func (p *Provider) Runtime() string {
	return p.PlatformRuntime
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

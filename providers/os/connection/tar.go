// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/cache"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	provider_tar "go.mondoo.com/cnquery/v10/providers/os/connection/tar"
	"go.mondoo.com/cnquery/v10/providers/os/fsutil"
	"go.mondoo.com/cnquery/v10/providers/os/id/containerid"
)

const (
	OPTION_FILE      = "path"
	FLATTENED_IMAGE  = "flattened_path"
	COMPRESSED_IMAGE = "compressed_path"
)

var _ shared.Connection = (*TarConnection)(nil)

type TarConnection struct {
	id        uint32
	asset     *inventory.Asset
	conf      *inventory.Config
	fetchFn   func() (string, error)
	fetchOnce sync.Once

	Fs      *provider_tar.FS
	CloseFN func()
	// fields are exposed since the tar backend is re-used for the docker backend
	PlatformKind         string
	PlatformRuntime      string
	PlatformIdentifier   string
	PlatformArchitecture string
	// optional metadata to store additional information
	Metadata struct {
		Name   string
		Labels map[string]string
	}
}

func (p *TarConnection) ID() uint32 {
	return p.id
}

func (p *TarConnection) Name() string {
	return string(shared.Type_Tar)
}

func (p *TarConnection) Type() shared.ConnectionType {
	return shared.Type_Tar
}

func (p *TarConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *TarConnection) Conf() *inventory.Config {
	return p.conf
}

func (c *TarConnection) Identifier() (string, error) {
	return c.PlatformIdentifier, nil
}

func (c *TarConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_FileSearch | shared.Capability_FindFile
}

func (p *TarConnection) RunCommand(command string) (*shared.Command, error) {
	res := shared.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (p *TarConnection) EnsureLoaded() {
	if p.fetchFn != nil {
		p.fetchOnce.Do(func() {
			f, err := p.fetchFn()
			if err != nil {
				log.Error().Err(err).Msg("tar> could not fetch tar file")
				return
			}
			if err := p.LoadFile(f); err != nil {
				log.Error().Err(err).Msg("tar> could not load tar file")
				return
			}
		})
	}
}

func (p *TarConnection) FileSystem() afero.Fs {
	p.EnsureLoaded()
	return p.Fs
}

func (c *TarConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	fs := c.FileSystem()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return shared.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*tar.Header); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}
	mode := stat.Mode()

	return shared.FileInfoDetails{
		Mode: shared.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (c *TarConnection) Close() {
	if c.CloseFN != nil {
		c.CloseFN()
	}
}

func (c *TarConnection) Load(stream io.Reader) error {
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

		path := provider_tar.Abs(h.Name)
		c.Fs.FileMap[path] = h
	}
	log.Debug().Int("files", len(c.Fs.FileMap)).Msg("tar> successfully loaded")
	return nil
}

func (c *TarConnection) LoadFile(path string) error {
	log.Debug().Str("path", path).Msg("tar> load tar file into backend")

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Load(f)
}

func (c *TarConnection) Kind() string {
	return c.PlatformKind
}

func (c *TarConnection) Runtime() string {
	return c.PlatformRuntime
}

func NewTarConnectionForContainer(id uint32, conf *inventory.Config, asset *inventory.Asset, img v1.Image) (*TarConnection, error) {
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	return &TarConnection{
		id:    id,
		asset: asset,
		Fs:    provider_tar.NewFs(""),
		fetchFn: func() (string, error) {
			err = cache.StreamToTmpFile(mutate.Extract(img), f)
			if err != nil {
				os.Remove(f.Name())
				return "", err
			}
			log.Debug().Msg("tar> extracted image to temporary file")
			asset.Connections[0].Options[FLATTENED_IMAGE] = f.Name()
			return f.Name(), nil
		},
		CloseFN: func() {
			log.Debug().Str("tar", f.Name()).Msg("tar> remove temporary tar file on connection close")
			os.Remove(f.Name())
		},
		PlatformKind:    conf.Type,
		PlatformRuntime: conf.Runtime,
		conf:            conf,
	}, nil
}

// TODO: this one is used by plain tar connection
func NewTarConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*TarConnection, error) {
	return NewWithClose(id, conf, asset, nil)
}

// Used with docker snapshots
// NewWithReader provides a tar provider from a container image stream
func NewWithReader(id uint32, conf *inventory.Config, asset *inventory.Asset, rc io.ReadCloser) (*TarConnection, error) {
	filename := ""
	if x, ok := rc.(*os.File); ok {
		filename = x.Name()
	} else {
		// cache file locally
		f, err := cache.RandomFile()
		if err != nil {
			return nil, err
		}

		// we return a pure tar image
		filename = f.Name()

		err = cache.StreamToTmpFile(rc, f)
		if err != nil {
			os.Remove(filename)
			return nil, err
		}
	}

	return NewWithClose(id, &inventory.Config{
		Type:    "tar",
		Runtime: "docker-image",
		Options: map[string]string{
			OPTION_FILE: filename,
		},
	}, asset, func() {
		log.Debug().Str("tar", filename).Msg("tar> remove temporary tar file on connection close")
		os.Remove(filename)
	})
}

func NewWithClose(id uint32, conf *inventory.Config, asset *inventory.Asset, closeFn func()) (*TarConnection, error) {
	if conf == nil || len(conf.Options[OPTION_FILE]) == 0 {
		return nil, errors.New("tar provider requires a valid tar file")
	}

	filename := conf.Options[OPTION_FILE]
	var identifier string

	// try to determine if the tar is a container image
	img, iErr := tarball.ImageFromPath(filename, nil)
	if iErr == nil {
		hash, err := img.Digest()
		if err != nil {
			return nil, err
		}
		identifier = containerid.MondooContainerImageID(hash.String())

		// we cache the flattened image locally
		c, err := newWithFlattenedImage(id, conf, asset, &img, closeFn)
		if err != nil {
			return nil, err
		}

		c.PlatformIdentifier = identifier
		return c, nil
	} else {
		hash, err := fsutil.LocalFileSha256(filename)
		if err != nil {
			return nil, err
		}
		identifier = "//platformid.api.mondoo.app/runtime/tar/hash/" + hash

		c := &TarConnection{
			id:              id,
			asset:           asset,
			Fs:              provider_tar.NewFs(filename),
			CloseFN:         closeFn,
			PlatformKind:    conf.Type,
			PlatformRuntime: conf.Runtime,
		}

		err = c.LoadFile(filename)
		if err != nil {
			log.Error().Err(err).Str("tar", filename).Msg("tar> could not load tar file")
			return nil, err
		}

		c.PlatformIdentifier = identifier
		return c, nil
	}
}

func newWithFlattenedImage(id uint32, conf *inventory.Config, asset *inventory.Asset, img *v1.Image, closeFn func()) (*TarConnection, error) {
	imageFilename := ""
	useCached := false
	if asset != nil && len(asset.Connections) > 0 {
		if x, ok := asset.Connections[0].Options[FLATTENED_IMAGE]; ok && x != "" {
			log.Debug().Str("tar", asset.Connections[0].Options[FLATTENED_IMAGE]).Msg("tar> use cached tar file")
			imageFilename = asset.Connections[0].Options[FLATTENED_IMAGE]
			useCached = true
		}
	}
	if !useCached {
		f, err := cache.RandomFile()
		if err != nil {
			return nil, err
		}
		imageFilename = f.Name()
		err = cache.StreamToTmpFile(mutate.Extract(*img), f)
		if err != nil {
			os.Remove(imageFilename)
			return nil, err
		}
	}

	c := &TarConnection{
		id:    id,
		asset: asset,
		Fs:    provider_tar.NewFs(imageFilename),
		CloseFN: func() {
			if closeFn != nil {
				closeFn()
			}
			// remove temporary file on stream close
			log.Debug().Str("tar", imageFilename).Msg("tar> remove temporary flattened image file on connection close")
			os.Remove(imageFilename)
		},
		PlatformKind:    conf.Type,
		PlatformRuntime: conf.Runtime,
		conf: &inventory.Config{
			Options: map[string]string{
				OPTION_FILE: imageFilename,
			},
		},
	}
	if asset != nil && len(asset.Connections) > 0 {
		if asset.Connections[0].Options == nil {
			asset.Connections[0].Options = map[string]string{}
		}
		asset.Connections[0].Options[FLATTENED_IMAGE] = imageFilename
	}

	err := c.LoadFile(imageFilename)
	if err != nil {
		log.Error().Err(err).Str("tar", imageFilename).Msg("tar> could not load tar file")
		os.Remove(imageFilename)
		return nil, err
	}

	return c, nil
}

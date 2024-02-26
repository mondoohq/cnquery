// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tar

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/fsutil"
)

const (
	OPTION_FILE = "path"
)

var (
	_ shared.Connection = (*Connection)(nil)
	_ plugin.Closer     = (*Connection)(nil)
)

type Connection struct {
	plugin.Connection
	asset     *inventory.Asset
	conf      *inventory.Config
	fetchFn   func() (string, error)
	fetchOnce sync.Once

	fs      *FS
	closeFN func()
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

func (p *Connection) Name() string {
	return string(shared.Type_Tar)
}

func (p *Connection) Type() shared.ConnectionType {
	return shared.Type_Tar
}

func (p *Connection) Asset() *inventory.Asset {
	return p.asset
}

func (p *Connection) Conf() *inventory.Config {
	return p.conf
}

func (c *Connection) Identifier() (string, error) {
	return c.PlatformIdentifier, nil
}

func (c *Connection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_FileSearch | shared.Capability_FindFile
}

func (p *Connection) RunCommand(command string) (*shared.Command, error) {
	res := shared.Command{Command: command, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ExitStatus: -1}
	return &res, nil
}

func (p *Connection) EnsureLoaded() {
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

func (p *Connection) FileSystem() afero.Fs {
	p.EnsureLoaded()
	return p.fs
}

func (c *Connection) FileInfo(path string) (shared.FileInfoDetails, error) {
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

func (c *Connection) Close() {
	if c.closeFN != nil {
		c.closeFN()
	}
}

func (c *Connection) Load(stream io.Reader) error {
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
		c.fs.FileMap[path] = h
	}
	log.Debug().Int("files", len(c.fs.FileMap)).Msg("tar> successfully loaded")
	return nil
}

func (c *Connection) LoadFile(path string) error {
	log.Debug().Str("path", path).Msg("tar> load tar file into backend")

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Load(f)
}

func (c *Connection) Kind() string {
	return c.PlatformKind
}

func (c *Connection) Runtime() string {
	return c.PlatformRuntime
}

type tarConnectionOptions struct {
	// provide a close function which is called when the connection is closed
	closeFn func()
	// function to fetch the tar file from a remote location on first access
	fetchFn func() (string, error)
}

type tarClientOption func(*tarConnectionOptions)

func WithCloseFn(closeFn func()) tarClientOption {
	return func(o *tarConnectionOptions) {
		o.closeFn = closeFn
	}
}

func WithFetchFn(fetchFn func() (string, error)) tarClientOption {
	return func(o *tarConnectionOptions) {
		o.fetchFn = fetchFn
	}
}

// NewConnection is opening a tar file and creating a new tar connection. The tar file is expected to be a valid
// tar file and contains a flattened file structure. Nested tar files as used in docker images are not supported and
// need to be extracted before using this connection.
func NewConnection(id uint32, conf *inventory.Config, asset *inventory.Asset, opts ...tarClientOption) (*Connection, error) {
	if conf == nil || len(conf.Options[OPTION_FILE]) == 0 {
		return nil, errors.New("tar provider requires a valid tar file")
	}

	filename := conf.Options[OPTION_FILE]
	var identifier string

	hash, err := fsutil.LocalFileSha256(filename)
	if err != nil {
		return nil, err
	}
	identifier = "//platformid.api.mondoo.app/runtime/tar/hash/" + hash

	params := &tarConnectionOptions{}
	for _, o := range opts {
		o(params)
	}

	c := &Connection{
		Connection:      plugin.NewConnection(id, asset),
		asset:           asset,
		fs:              NewFs(filename),
		closeFN:         params.closeFn,
		fetchFn:         params.fetchFn,
		PlatformKind:    conf.Type,
		PlatformRuntime: conf.Runtime,
		conf:            conf,
	}

	// if no fetch function is provided, we use the local file as source which allows to use load the tar file
	// from a local file system asynchronous.
	if params.fetchFn == nil {
		c.fetchFn = func() (string, error) {
			return filename, nil
		}
	}

	c.PlatformIdentifier = identifier
	return c, nil
}

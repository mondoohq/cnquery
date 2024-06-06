// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package fs

import (
	"errors"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/fs"
)

var (
	_ shared.Connection = (*FileSystemConnection)(nil)
	_ plugin.Closer     = (*FileSystemConnection)(nil)
)

func NewFileSystemConnectionWithClose(id uint32, conf *inventory.Config, asset *inventory.Asset, closeFN func()) (*FileSystemConnection, error) {
	path, ok := conf.Options["path"]
	if !ok {
		// fallback to host + path option
		path = conf.Host + conf.Path
	}

	if path == "" {
		return nil, errors.New("missing filesystem mount path, use 'path' option")
	}

	log.Debug().Str("path", path).Msg("load filesystem")

	return &FileSystemConnection{
		Connection:   plugin.NewConnection(id, asset),
		Conf:         conf,
		asset:        asset,
		MountedDir:   path,
		closeFN:      closeFN,
		tcPlatformId: conf.PlatformId,
		fs:           fs.NewMountedFs(path),
	}, nil
}

func NewConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*FileSystemConnection, error) {
	return NewFileSystemConnectionWithClose(id, conf, asset, nil)
}

type FileSystemConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	MountedDir   string
	fs           afero.Fs
	tcPlatformId string
	closeFN      func()
}

func (c *FileSystemConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, plugin.ErrRunCommandNotImplemented
}

func (c *FileSystemConnection) FileSystem() afero.Fs {
	if c.fs == nil {
		c.fs = fs.NewMountedFs(c.MountedDir)
	}
	return c.fs
}

func (c *FileSystemConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	fs := c.FileSystem()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return shared.FileInfoDetails{}, err
	}

	uid, gid := c.fileowner(stat)

	mode := stat.Mode()
	return shared.FileInfoDetails{
		Mode: shared.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
		Path: filepath.Join(c.MountedDir, path),
	}, nil
}

func (c *FileSystemConnection) Close() {
	if c.closeFN != nil {
		c.closeFN()
	}
}

func (c *FileSystemConnection) Capabilities() shared.Capabilities {
	return shared.Capability_FileSearch | shared.Capability_File | shared.Capability_FindFile
}

func (c *FileSystemConnection) Identifier() (string, error) {
	if c.tcPlatformId == "" {
		return "", errors.New("no platform id provided")
	}
	return c.tcPlatformId, nil
}

func (c *FileSystemConnection) Name() string {
	return string(shared.Type_FileSystem)
}

func (c *FileSystemConnection) Type() shared.ConnectionType {
	return shared.Type_FileSystem
}

func (c *FileSystemConnection) Asset() *inventory.Asset {
	return c.asset
}

func (p *FileSystemConnection) UpdateAsset(asset *inventory.Asset) {
	p.asset = asset
}

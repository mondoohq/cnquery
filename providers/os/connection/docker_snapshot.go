// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"os"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/os/connection/container/cache"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
)

var _ shared.Connection = &DockerSnapshotConnection{}

type DockerSnapshotConnection struct {
	TarConnection
}

func NewDockerSnapshotConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*DockerSnapshotConnection, error) {
	tarConnection, err := NewWithClose(id, conf, asset, func() {})
	if err != nil {
		return nil, err
	}

	// FIXME: ??? use NewFromDockerEngine

	return &DockerSnapshotConnection{*tarConnection}, nil
}

// NewFromDockerEngine creates a snapshot for a docker engine container and opens it
func NewFromDockerEngine(id uint32, conf *inventory.Config, asset *inventory.Asset) (*DockerSnapshotConnection, error) {
	// cache container on local disk
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	err = ExportSnapshot(conf.Host, f)
	if err != nil {
		return nil, err
	}

	tarConnection, err := NewWithClose(id, &inventory.Config{
		Type: "tar",
		Options: map[string]string{
			OPTION_FILE: f.Name(),
		},
	}, asset, func() {
		// remove temporary file on stream close
		os.Remove(f.Name())
	})
	if err != nil {
		return nil, err
	}

	return &DockerSnapshotConnection{*tarConnection}, nil
}

// ExportSnapshot exports a given container from docker engine to a tar file
func ExportSnapshot(containerid string, f *os.File) error {
	dc, err := GetDockerClient()
	if err != nil {
		return err
	}

	rc, err := dc.ContainerExport(context.Background(), containerid)
	if err != nil {
		return err
	}

	return cache.StreamToTmpFile(rc, f)
}

func (p *DockerSnapshotConnection) ID() uint32 {
	return p.id
}

func (p *DockerSnapshotConnection) Name() string {
	return string(shared.Type_DockerSnapshot)
}

func (p *DockerSnapshotConnection) Type() shared.ConnectionType {
	return shared.Type_DockerSnapshot
}

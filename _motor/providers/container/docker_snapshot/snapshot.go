// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker_snapshot

import (
	"context"
	"os"

	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/container/cache"
	"go.mondoo.com/cnquery/motor/providers/container/docker_engine"
	"go.mondoo.com/cnquery/motor/providers/tar"
)

type DockerSnapshotProvider struct {
	tar.Provider
}

// NewFromDockerEngine creates a snapshot for a docker engine container and opens it
func NewFromDockerEngine(containerid string) (*DockerSnapshotProvider, error) {
	// cache container on local disk
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	err = ExportSnapshot(containerid, f)
	if err != nil {
		return nil, err
	}

	tarProvider, err := tar.NewWithClose(&providers.Config{
		Backend: providers.ProviderType_TAR,
		Options: map[string]string{
			tar.OPTION_FILE: f.Name(),
		},
	}, func() {
		// remove temporary file on stream close
		os.Remove(f.Name())
	})
	if err != nil {
		return nil, err
	}

	return &DockerSnapshotProvider{Provider: *tarProvider}, nil
}

// ExportSnapshot exports a given container from docker engine to a tar file
func ExportSnapshot(containerid string, f *os.File) error {
	dc, err := docker_engine.GetDockerClient()
	if err != nil {
		return err
	}

	rc, err := dc.ContainerExport(context.Background(), containerid)
	if err != nil {
		return err
	}

	return cache.StreamToTmpFile(rc, f)
}

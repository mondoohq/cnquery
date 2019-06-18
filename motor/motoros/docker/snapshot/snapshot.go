package snapshot

import (
	"context"
	"os"

	"go.mondoo.io/mondoo/motor/motoros/docker/cache"
	"go.mondoo.io/mondoo/motor/motoros/docker/docker_engine"
	"go.mondoo.io/mondoo/motor/motoros/tar"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func NewFromDockerEngine(containerid string) (types.Transport, error) {
	// cache container on local disk
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	err = Export(containerid, f)
	if err != nil {
		return nil, err
	}

	return tar.NewWithClose(&types.Endpoint{Path: f.Name()}, func() {
		// remove temporary file on stream close
		os.Remove(f.Name())
	})
}

func NewFromDirectory(path string) (types.Transport, error) {
	return tar.New(&types.Endpoint{Path: path})
}

// exports a given container from docker engine to a tar file
func Export(containerid string, f *os.File) error {
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

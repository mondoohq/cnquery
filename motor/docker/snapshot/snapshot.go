package snapshot

import (
	"context"
	"os"

	"go.mondoo.io/mondoo/motor/docker/cache"
	"go.mondoo.io/mondoo/motor/docker/docker_engine"
	"go.mondoo.io/mondoo/motor/tar"
	"go.mondoo.io/mondoo/motor/types"
)

func NewFromDockerEngine(containerid string) (types.Transport, error) {
	// cache container on local disk
	filename := cache.RandomFile()
	err := Export(containerid, filename)
	if err != nil {
		return nil, err
	}

	return tar.NewWithClose(&types.Endpoint{Path: filename}, func() {
		// remove temporary file on stream close
		os.Remove(filename)
	})
}

func NewFromDirectory(path string) (types.Transport, error) {
	return tar.New(&types.Endpoint{Path: path})
}

// exports a given container from docker engine to a tar file
func Export(containerid string, filename string) error {
	dc, err := docker_engine.GetDockerClient()
	if err != nil {
		return err
	}

	rc, err := dc.ContainerExport(context.Background(), containerid)
	if err != nil {
		return err
	}

	cache.StreamToTmpFile(rc, filename)
	return nil
}

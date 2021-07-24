package image

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/container/cache"
	"go.mondoo.io/mondoo/motor/transports/tar"
)

type ContainerImageTransport struct {
	tar.Transport
}

// New provides a container image stream
func New(rc io.ReadCloser) (*ContainerImageTransport, error) {
	// we cache the flattened image locally
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	err = cache.StreamToTmpFile(rc, f)
	if err != nil {
		return nil, err
	}

	// we return a pure tar image
	filename := f.Name()

	tarTransport, err := tar.NewWithClose(&transports.TransportConfig{
		Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
		Runtime: transports.RUNTIME_DOCKER_IMAGE,
		Options: map[string]string{
			tar.OPTION_FILE: filename,
		},
	}, func() {
		// remove temporary file on stream close
		os.Remove(filename)
	})
	if err != nil {
		return nil, err
	}

	return &ContainerImageTransport{Transport: *tarTransport}, nil
}

func ImageToTar(filename string, img v1.Image, baseName, imgName, tagName string) error {
	imgTag := fmt.Sprintf("%s/%s:%s", baseName, imgName, tagName)
	tag, err := name.NewTag(imgTag, name.WeakValidation)
	if err != nil {
		return errors.New(fmt.Sprintf("parsing tag %q: %v", imgTag, err))
	}
	return tarball.WriteToFile(filename, tag, img)
}

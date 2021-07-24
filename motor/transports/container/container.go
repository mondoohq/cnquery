package container

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	docker_discovery "go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid/containerid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/container/docker_engine"
	"go.mondoo.io/mondoo/motor/transports/container/docker_snapshot"
	"go.mondoo.io/mondoo/motor/transports/container/image"
)

type ContainerTransport interface {
	transports.Transport
	Identifier() string
	Labels() map[string]string
	PlatformName() string
}

func NewContainerRegistryImage(tc *transports.TransportConfig) (ContainerTransport, error) {
	// load container image from remote directoryload tar file into backend
	ref, err := name.ParseReference(tc.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(tc.Insecure)}
		if len(tc.BearerToken) > 0 {
			log.Debug().Msg("enable bearer authentication for image")
			registryOpts = append(registryOpts, image.WithAuthenticator(&authn.Bearer{Token: tc.BearerToken}))
		}

		// image.WithAuthenticator()
		img, rc, err := image.LoadImageFromRegistry(ref, registryOpts...)
		if err != nil {
			return nil, err
		}

		var identifier string
		hash, err := img.Digest()
		if err == nil {
			identifier = containerid.MondooContainerImageID(hash.String())
		}

		transport, err := image.New(rc)
		if err != nil {
			return nil, err
		}
		transport.PlatformIdentifier = identifier
		transport.Metadata.Name = containerid.ShortContainerImageID(hash.String())

		return transport, err
	}
	log.Debug().Str("image", tc.Host).Msg("Could not detect a valid repository url")
	return nil, err
}

func NewDockerEngineContainer(tc *transports.TransportConfig) (ContainerTransport, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, err
	}

	ci, err := ded.ContainerInfo(tc.Host)
	if err != nil {
		return nil, err
	}

	if ci.Running {
		log.Debug().Msg("found running container " + ci.ID)
		transport, err := docker_engine.New(ci.ID)
		if err != nil {
			return nil, err
		}
		transport.PlatformIdentifier = containerid.MondooContainerID(ci.ID)
		transport.Metadata.Name = containerid.ShortContainerImageID(ci.ID)
		transport.Metadata.Labels = ci.Labels
		return transport, nil
	} else {
		log.Debug().Msg("found stopped container " + ci.ID)
		transport, err := docker_snapshot.NewFromDockerEngine(ci.ID)
		if err != nil {
			return nil, err
		}
		transport.PlatformIdentifier = containerid.MondooContainerID(ci.ID)
		transport.Metadata.Name = containerid.ShortContainerImageID(ci.ID)
		transport.Metadata.Labels = ci.Labels
		return transport, nil
	}
}

func NewDockerEngineImage(endpoint *transports.TransportConfig) (ContainerTransport, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, err
	}

	ii, err := ded.ImageInfo(endpoint.Host)
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("found docker engine image " + ii.ID)
	img, rc, err := image.LoadImageFromDockerEngine(ii.ID)
	if err != nil {
		return nil, err
	}

	var identifier string
	hash, err := img.Digest()
	if err == nil {
		identifier = containerid.MondooContainerImageID(hash.String())
	}

	transport, err := image.New(rc)
	if err != nil {
		return nil, err
	}
	transport.PlatformIdentifier = identifier
	transport.Metadata.Name = ii.Name
	transport.Metadata.Labels = ii.Labels
	return transport, nil
}

// check if the tar is an image or container
func NewContainerTar(endpoint *transports.TransportConfig) (ContainerTransport, error) {
	log.Debug().Msg("found local docker/image file")

	// try to load docker image tarball
	img, err := tarball.ImageFromPath(endpoint.Host, nil)
	if err == nil {
		log.Debug().Msg("detected docker image")
		var identifier string

		hash, err := img.Digest()
		if err == nil {
			identifier = containerid.MondooContainerImageID(hash.String())
		} else {
			log.Warn().Err(err).Msg("could not determine platform id")
		}

		rc := mutate.Extract(img)
		transport, err := image.New(rc)
		if err != nil {
			return nil, err
		}
		transport.PlatformIdentifier = identifier
		return transport, err
	}

	log.Debug().Msg("detected docker container snapshot")

	// generate sha sum of tar file
	f, err := os.Open(endpoint.Host)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read container tar to generate hash")
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrap(err, "cannot read container tar to generate hash")
	}

	hash := hex.EncodeToString(h.Sum(nil))

	transport, err := docker_snapshot.NewFromFile(endpoint.Host)
	transport.PlatformIdentifier = "//platformid.api.mondoo.app/runtime/docker/snapshot/" + hash
	return transport, err
}

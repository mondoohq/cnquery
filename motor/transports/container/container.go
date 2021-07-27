package container

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	docker_discovery "go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid/containerid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/container/docker_engine"
	"go.mondoo.io/mondoo/motor/transports/container/docker_snapshot"
	"go.mondoo.io/mondoo/motor/transports/container/image"
	"go.mondoo.io/mondoo/motor/transports/tar"
)

type ContainerTransport interface {
	transports.Transport
	Identifier() string
	Labels() map[string]string
	PlatformName() string
}

// NewContainerRegistryImage loads a container image from a remote registry
func NewContainerRegistryImage(tc *transports.TransportConfig) (ContainerTransport, error) {
	ref, err := name.ParseReference(tc.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(tc.Insecure)}
		if len(tc.Options["bearer"]) > 0 {
			log.Debug().Msg("enable bearer authentication for image")
			registryOpts = append(registryOpts, image.WithAuthenticator(&authn.Bearer{Token: tc.Options["bearer"]}))
		}

		img, rc, err := image.LoadImageFromRegistry(ref, registryOpts...)
		if err != nil {
			return nil, err
		}

		var identifier string
		hash, err := img.Digest()
		if err == nil {
			identifier = containerid.MondooContainerImageID(hash.String())
		}

		transport, err := tar.NewWithReader(rc, nil)
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

	transport, err := tar.NewWithReader(rc, nil)
	if err != nil {
		return nil, err
	}
	transport.PlatformIdentifier = identifier
	transport.Metadata.Name = ii.Name
	transport.Metadata.Labels = ii.Labels
	return transport, nil
}

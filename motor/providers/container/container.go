package container

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	docker_discovery "go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid/containerid"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/container/auth"
	"go.mondoo.io/mondoo/motor/providers/container/docker_engine"
	"go.mondoo.io/mondoo/motor/providers/container/docker_snapshot"
	"go.mondoo.io/mondoo/motor/providers/container/image"
	"go.mondoo.io/mondoo/motor/providers/tar"
)

type ContainerProvider interface {
	providers.Transport
	providers.TransportPlatformIdentifier
	Labels() map[string]string
	PlatformName() string
}

// NewContainerRegistryImage loads a container image from a remote registry
func NewContainerRegistryImage(tc *providers.Config) (ContainerProvider, error) {
	ref, err := name.ParseReference(tc.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(tc.Insecure)}
		remoteOpts := auth.AuthOption(tc.Credentials)
		for i := range remoteOpts {
			registryOpts = append(registryOpts, remoteOpts[i])
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

		// set the platform architecture using the image configuration
		imgConfig, err := img.ConfigFile()
		if err == nil {
			transport.PlatformArchitecture = imgConfig.Architecture
		}

		return transport, err
	}
	log.Debug().Str("image", tc.Host).Msg("Could not detect a valid repository url")
	return nil, err
}

func NewDockerEngineContainer(tc *providers.Config) (ContainerProvider, error) {
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
		p, err := docker_engine.New(ci.ID)
		if err != nil {
			return nil, err
		}
		p.PlatformIdentifier = containerid.MondooContainerID(ci.ID)
		p.Metadata.Name = containerid.ShortContainerImageID(ci.ID)
		p.Metadata.Labels = ci.Labels
		return p, nil
	} else {
		log.Debug().Msg("found stopped container " + ci.ID)
		p, err := docker_snapshot.NewFromDockerEngine(ci.ID)
		if err != nil {
			return nil, err
		}
		p.PlatformIdentifier = containerid.MondooContainerID(ci.ID)
		p.Metadata.Name = containerid.ShortContainerImageID(ci.ID)
		p.Metadata.Labels = ci.Labels
		return p, nil
	}
}

func NewDockerEngineImage(endpoint *providers.Config) (ContainerProvider, error) {
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

	p, err := tar.NewWithReader(rc, nil)
	if err != nil {
		return nil, err
	}
	p.PlatformIdentifier = identifier
	p.Metadata.Name = ii.Name
	p.Metadata.Labels = ii.Labels
	return p, nil
}

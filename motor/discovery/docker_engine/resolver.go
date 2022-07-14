package docker_engine

import (
	"os"

	"go.mondoo.io/mondoo/motor/discovery/credentials"

	"github.com/cockroachdb/errors"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/container_registry"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/tar"
)

const (
	DiscoveryAll              = "all"
	DiscoveryContainerRunning = "container"
	DiscoveryContainerImages  = "container-images"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Docker Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryContainerRunning, DiscoveryContainerImages}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	if tc == nil {
		return nil, errors.New("no transport configuration found")
	}

	// check if we have a tar as input
	// detect if the tar is a container image format -> container image
	// or a container snapshot format -> container snapshot
	if tc.Backend == transports.TransportBackend_CONNECTION_TAR {

		if tc.Options == nil || tc.Options["file"] == "" {
			return nil, errors.New("could not find the tar file")
		}

		filename := tc.Options["file"]

		// check if we are pointing to a local tar file
		_, err := os.Stat(filename)
		if err != nil {
			return nil, errors.New("could not find the tar file: " + filename)
		}

		// Tar container can be an image or a snapshot
		resolvedAsset := &asset.Asset{
			Name:        filename,
			Connections: []*transports.TransportConfig{tc},
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
				Runtime: transports.RUNTIME_DOCKER_IMAGE,
			},
			State: asset.State_STATE_ONLINE,
		}

		// determine platform identifier
		identifier, err := tar.PlatformID(filename)
		if err != nil {
			return nil, err
		}

		resolvedAsset.PlatformIds = []string{identifier}

		return []*asset.Asset{resolvedAsset}, nil
	}

	ded, dockerEngErr := NewDockerEngineDiscovery()
	// we do not fail here, since we pull the image from upstream if its is an image without the need for docker

	if tc.Backend == transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER {
		if dockerEngErr != nil {
			return nil, errors.Wrap(dockerEngErr, "cannot connect to docker engine to fetch the container")
		}
		resolvedAsset, err := r.container(tc, ded)
		if err != nil {
			return nil, err
		}

		return []*asset.Asset{resolvedAsset}, nil
	}

	if tc.Backend == transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE {
		// NOTE, we ignore dockerEngErr here since we fallback to pulling the images directly
		resolvedAssets, err := r.images(tc, ded, cfn, sfn)
		if err != nil {
			return nil, err
		}
		return resolvedAssets, nil
	}

	// check if we should do a discovery
	if tc.Host == "" {
		return DiscoverDockerEngineAssets(tc)
	}

	// if we are here, the user has not specified the direct target, we need to search for it
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	// 1. check if we have a container id
	//    check if the container is running -> docker engine
	//    check if the container is stopped -> container snapshot
	// 3. check if we have an image id -> container image
	// 4. check if we have a descriptor for a registry -> container image
	log.Debug().Str("docker", tc.Host).Msg("try to resolve the container or image source")

	if dockerEngErr == nil {
		containerAsset, err := r.container(tc, ded)
		if err == nil {
			return []*asset.Asset{containerAsset}, nil
		}
	}

	containerImageAssets, err := r.images(tc, ded, cfn, sfn)
	if err == nil {
		return containerImageAssets, nil
	}

	// if we reached here, we assume we have a name of an image or container from a registry
	return nil, errors.Wrap(err, "could not find the container reference")
}

func (k *Resolver) container(tc *transports.TransportConfig, ded *dockerEngineDiscovery) (*asset.Asset, error) {
	ci, err := ded.ContainerInfo(tc.Host)
	if err != nil {
		return nil, err
	}

	tc.Backend = transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER
	return &asset.Asset{
		Name:        ci.Name,
		Connections: []*transports.TransportConfig{tc},
		PlatformIds: []string{ci.PlatformID},
		Platform: &platform.Platform{
			Kind:    transports.Kind_KIND_CONTAINER,
			Runtime: transports.RUNTIME_DOCKER_CONTAINER,
		},
		State:  asset.State_STATE_ONLINE,
		Labels: ci.Labels,
	}, nil
}

func (k *Resolver) images(tc *transports.TransportConfig, ded *dockerEngineDiscovery, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn) ([]*asset.Asset, error) {
	// if we have a docker engine available, try to fetch it from there
	if ded != nil {
		ii, err := ded.ImageInfo(tc.Host)
		if err == nil {
			tc.Backend = transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE
			return []*asset.Asset{{
				Name:        ii.Name,
				Connections: []*transports.TransportConfig{tc},
				PlatformIds: []string{ii.PlatformID},
				Platform: &platform.Platform{
					Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
					Runtime: transports.RUNTIME_DOCKER_IMAGE,
				},
				State:  asset.State_STATE_ONLINE,
				Labels: ii.Labels,
			}}, nil
		}

	}

	// otherwise try to fetch the image from upstream
	log.Debug().Msg("try to download the image from docker registry")
	_, err := name.ParseReference(tc.Host, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	// switch to container registry resolver since docker is not installed
	rr := container_registry.Resolver{
		NoStrictValidation: true,
	}
	return rr.Resolve(tc, cfn, sfn)
}

func DiscoverDockerEngineAssets(tc *transports.TransportConfig) ([]*asset.Asset, error) {
	log.Debug().Msg("start discovery for docker engine")
	// we use generic `container` and `container-images` options to avoid the requirement for the user to know if
	// the system is using docker or podman locally
	assetList := []*asset.Asset{}

	// discover running container: container
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerRunning) {
		ded, err := NewDockerEngineDiscovery()
		if err != nil {
			return nil, err
		}

		containerAssets, err := ded.ListContainer()
		if err != nil {
			return nil, err
		}
		log.Info().Int("images", len(containerAssets)).Msg("running container search completed")
		assetList = append(assetList, containerAssets...)
	}

	// discover container images: container-images
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerImages) {
		ded, err := NewDockerEngineDiscovery()
		if err != nil {
			return nil, err
		}

		containerImageAssets, err := ded.ListImages()
		if err != nil {
			return nil, err
		}
		log.Info().Int("images", len(containerImageAssets)).Msg("running container images search completed")
		assetList = append(assetList, containerImageAssets...)
	}
	return assetList, nil
}

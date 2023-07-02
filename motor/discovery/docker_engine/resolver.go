package docker_engine

import (
	"context"
	"os"

	"errors"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/discovery/container_registry"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/tar"
	"go.mondoo.com/cnquery/motor/vault"
)

const (
	DiscoveryContainerRunning = "container"
	DiscoveryContainerImages  = "container-images"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Docker Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryContainerRunning, DiscoveryContainerImages}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	if pCfg == nil {
		return nil, errors.New("no provider configuration found")
	}

	// check if we have a tar as input
	// detect if the tar is a container image format -> container image
	// or a container snapshot format -> container snapshot
	if pCfg.Backend == providers.ProviderType_TAR {

		if pCfg.Options == nil || pCfg.Options["file"] == "" {
			return nil, errors.New("could not find the tar file")
		}

		filename := pCfg.Options["file"]

		// check if we are pointing to a local tar file
		_, err := os.Stat(filename)
		if err != nil {
			return nil, errors.New("could not find the tar file: " + filename)
		}

		// Tar container can be an image or a snapshot
		resolvedAsset := &asset.Asset{
			Name:        filename,
			Connections: []*providers.Config{pCfg},
			Platform: &platform.Platform{
				Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
				Runtime: providers.RUNTIME_DOCKER_IMAGE,
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

	if pCfg.Backend == providers.ProviderType_DOCKER_ENGINE_CONTAINER {
		if dockerEngErr != nil {
			return nil, errors.Join(dockerEngErr, errors.New("cannot connect to docker engine to fetch the container"))
		}
		resolvedAsset, err := r.container(ctx, root, pCfg, ded)
		if err != nil {
			return nil, err
		}

		return []*asset.Asset{resolvedAsset}, nil
	}

	if pCfg.Backend == providers.ProviderType_DOCKER_ENGINE_IMAGE {
		// NOTE, we ignore dockerEngErr here since we fallback to pulling the images directly
		resolvedAssets, err := r.images(ctx, root, pCfg, ded, credsResolver, sfn)
		if err != nil {
			return nil, err
		}
		return resolvedAssets, nil
	}

	// check if we should do a discovery
	if pCfg.Host == "" {
		return DiscoverDockerEngineAssets(pCfg)
	}

	// if we are here, the user has not specified the direct target, we need to search for it
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	// 1. check if we have a container id
	//    check if the container is running -> docker engine
	//    check if the container is stopped -> container snapshot
	// 3. check if we have an image id -> container image
	// 4. check if we have a descriptor for a registry -> container image
	log.Debug().Str("docker", pCfg.Host).Msg("try to resolve the container or image source")

	if dockerEngErr == nil {
		containerAsset, err := r.container(ctx, root, pCfg, ded)
		if err == nil {
			return []*asset.Asset{containerAsset}, nil
		}
	}

	containerImageAssets, err := r.images(ctx, root, pCfg, ded, credsResolver, sfn)
	if err == nil {
		return containerImageAssets, nil
	}

	// if we reached here, we assume we have a name of an image or container from a registry
	return nil, errors.Join(err, errors.New("could not find the container reference"))
}

func (k *Resolver) container(ctx context.Context, root *asset.Asset, pCfg *providers.Config, ded *dockerEngineDiscovery) (*asset.Asset, error) {
	ci, err := ded.ContainerInfo(pCfg.Host)
	if err != nil {
		return nil, err
	}

	pCfg.Backend = providers.ProviderType_DOCKER_ENGINE_CONTAINER

	// TODO: how do we know we're not connecting to docker over
	// the network and LOCAL_OS is correct
	relatedAssets := []*asset.Asset{
		{
			Connections: []*providers.Config{
				{
					Backend: providers.ProviderType_LOCAL_OS,
				},
			},
		},
	}

	return &asset.Asset{
		Name:        ci.Name,
		Connections: []*providers.Config{pCfg},
		PlatformIds: []string{ci.PlatformID},
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_CONTAINER,
			Runtime: providers.RUNTIME_DOCKER_CONTAINER,
		},
		State:         asset.State_STATE_ONLINE,
		Labels:        ci.Labels,
		RelatedAssets: relatedAssets,
	}, nil
}

func (k *Resolver) images(ctx context.Context, root *asset.Asset, pCfg *providers.Config, ded *dockerEngineDiscovery, credsResolver vault.Resolver, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	// if we have a docker engine available, try to fetch it from there
	if ded != nil {
		ii, err := ded.ImageInfo(pCfg.Host)
		if err == nil {
			pCfg.Backend = providers.ProviderType_DOCKER_ENGINE_IMAGE
			return []*asset.Asset{{
				Name:        ii.Name,
				Connections: []*providers.Config{pCfg},
				PlatformIds: []string{ii.PlatformID},
				Platform: &platform.Platform{
					Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
					Runtime: providers.RUNTIME_DOCKER_IMAGE,
				},
				State:  asset.State_STATE_ONLINE,
				Labels: ii.Labels,
			}}, nil
		}

	}

	// otherwise try to fetch the image from upstream
	log.Debug().Msg("try to download the image from docker registry")
	_, err := name.ParseReference(pCfg.Host, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	// switch to container registry resolver since docker is not installed
	rr := container_registry.Resolver{
		NoStrictValidation: true,
	}
	return rr.Resolve(ctx, root, pCfg, credsResolver, sfn)
}

func DiscoverDockerEngineAssets(pCfg *providers.Config) ([]*asset.Asset, error) {
	log.Debug().Msg("start discovery for docker engine")
	// we use generic `container` and `container-images` options to avoid the requirement for the user to know if
	// the system is using docker or podman locally
	assetList := []*asset.Asset{}

	// discover running container: container
	if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryContainerRunning) {
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
	if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryContainerImages) {
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

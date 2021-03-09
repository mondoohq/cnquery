package docker_engine

import (
	"strings"

	"os"

	"github.com/cockroachdb/errors"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"

	"go.mondoo.io/mondoo/motor/transports"
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

// When we talk about Docker, users think at leasst of 3 different things:
// - container runtime (e.g. docker engine)
// - container image (eg. from docker engine or registry)
// - container tar snapshot
//
// Docker made a very good job in abstracting the problem away from the user
// so that he normally does not think about the distinction. But we need to
// think about those aspects since all those need a different implementation and
// handling.
//
// The user wants and needs an easy way to point to those endpoints:
//
// # registry images
// -t docker://gcr.io/project/image@sha256:label
// -t docker://index.docker.io/project/image:label
//
// # docker daemon
// -t docker://id -> image
// -t docker://id -> container
//
// # local directory
// -t docker+tar:///path/link_to_image_archive.tar -> Docker Image
// -t docker+tar:///path/link_to_image_archive2.tar -> OCI
// -t docker+tar:///path/link_to_container.tar
func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	if strings.HasPrefix(url, transports.SCHEME_DOCKER+"://") {
		tc := &transports.TransportConfig{
			Backend: transports.TransportBackend_CONNECTION_DOCKER,
			Host:    strings.Replace(url, transports.SCHEME_DOCKER+"://", "", 1),
		}

		for i := range opts {
			opts[i](tc)
		}
		return tc, nil
	} else if strings.HasPrefix(url, transports.SCHEME_DOCKER_IMAGE+"://") {
		tc := &transports.TransportConfig{
			Backend: transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE,
			Host:    strings.Replace(url, transports.SCHEME_DOCKER_IMAGE+"://", "", 1),
		}
		for i := range opts {
			opts[i](tc)
		}
		return tc, nil
	} else if strings.HasPrefix(url, transports.SCHEME_DOCKER_CONTAINER+"://") {
		tc := &transports.TransportConfig{
			Backend: transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER,
			Host:    strings.Replace(url, transports.SCHEME_DOCKER_CONTAINER+"://", "", 1),
		}
		for i := range opts {
			opts[i](tc)
		}
		return tc, nil
	} else if strings.HasPrefix(url, transports.SCHEME_DOCKER_TAR+"://") {
		tc := &transports.TransportConfig{
			Backend: transports.TransportBackend_CONNECTION_DOCKER_ENGINE_TAR,
			Host:    strings.Replace(url, transports.SCHEME_DOCKER_TAR+"://", "", 1),
		}

		for i := range opts {
			opts[i](tc)
		}
		return tc, nil
	}
	return nil, errors.New("could not find the container reference")
}

func (r *Resolver) Resolve(tc *transports.TransportConfig) ([]*asset.Asset, error) {
	if tc == nil {
		return nil, errors.New("no transport configuration found")
	}

	// check if we have a tar as input
	// detect if the tar is a container image format -> container image
	// or a container snapshot format -> container snapshot
	if tc.Backend == transports.TransportBackend_CONNECTION_DOCKER_ENGINE_TAR {
		// check if we are pointing to a local tar file
		_, err := os.Stat(tc.Host)
		if err != nil {
			return nil, errors.New("could not find the tar file: " + tc.Host)
		}
		log.Debug().Msg("detected local container tar file")

		// Tar container can be an image or a snapshot
		resolvedAsset := &asset.Asset{
			Name:        tc.Host,
			Connections: []*transports.TransportConfig{tc},
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
				Runtime: transports.RUNTIME_DOCKER_IMAGE,
			},
		}
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
		// NOTE, we ignore dockerEngErr here since we fallback to pulling the image directly
		resolvedAsset, err := r.image(tc, ded)
		if err != nil {
			return nil, err
		}
		return []*asset.Asset{resolvedAsset}, nil
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

	containerImageAsset, err := r.image(tc, ded)
	if err == nil {
		return []*asset.Asset{containerImageAsset}, nil
	}

	// if we reached here, we assume we have a name of an image or container from a registry
	return nil, errors.New("could not find the container reference")
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
		PlatformIDs: []string{ci.PlatformID},
		Platform: &platform.Platform{
			Kind:    transports.Kind_KIND_CONTAINER,
			Runtime: transports.RUNTIME_DOCKER_CONTAINER,
		},
	}, nil
}

func (k *Resolver) image(tc *transports.TransportConfig, ded *dockerEngineDiscovery) (*asset.Asset, error) {
	// if we have a docker engine available, try to fetch it from there
	if ded != nil {
		ii, err := ded.ImageInfo(tc.Host)
		if err == nil {
			tc.Backend = transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE
			return &asset.Asset{
				Name:        ii.Name,
				Connections: []*transports.TransportConfig{tc},
				PlatformIDs: []string{ii.PlatformID},
				Platform: &platform.Platform{
					Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
					Runtime: transports.RUNTIME_DOCKER_IMAGE,
				},
			}, nil
		}
	}

	log.Debug().Msg("try to download the image from docker registry")
	_, err := name.ParseReference(tc.Host, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	tc.Backend = transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY
	return &asset.Asset{
		Name: tc.Host,
		// PlatformIDs: []string{}, // we cannot determine the id here
		Connections: []*transports.TransportConfig{tc},
		Platform: &platform.Platform{
			Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
			Runtime: transports.RUNTIME_DOCKER_REGISTRY,
		},
	}, nil
}

func DiscoverDockerEngineAssets(tc *transports.TransportConfig) ([]*asset.Asset, error) {
	// we use generic `container` and `container-images` options to avoid the requirement for the user to know if
	// the system is using docker or podman locally
	assetList := []*asset.Asset{}

	// discover running container: container:true
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

	// discover container images: container-images:true
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryContainerImages) {
		ded, err := NewDockerEngineDiscovery()
		if err != nil {
			return nil, err
		}

		containerImageAssets, err := ded.ListImages()
		if err != nil {
			return nil, err
		}
		log.Info().Int("images", len(containerImageAssets)).Msg("running container image search completed")
		assetList = append(assetList, containerImageAssets...)
	}
	return assetList, nil
}

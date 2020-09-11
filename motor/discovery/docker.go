package discovery

import (
	"strings"

	"os"

	"github.com/cockroachdb/errors"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"

	docker_discovery "go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/transports"
)

type DockerInfo struct {
	Name       string
	Identifier string
	Labels     map[string]string
}

type dockerResolver struct{}

// When we talk about Docker, users think at leasst of 3 different things:
// - container runtime (e.g. docker engine)
// - container image (eg. from docker engine or registry)
// - container snapshot
//
// Docker made a very good job in abstracting the problem away from the user
// so that he normally does not think about the distinction. But we need to
// think about those aspects since all those need a different implementation and
// handling.
//
// The user wants and needs an easy way to point to those endpoints:
//
// # registries
// -t docker://gcr.io/project/image@sha256:label
// -t docker://index.docker.io/project/image:label
//
// # docker daemon
// -t docker://id -> image
// -t docker://id -> container
//
// # local directory
// -t docker:///path/link_to_image_archive.tar -> Docker Image
// -t docker:///path/link_to_image_archive2.tar -> OCI
// -t docker:///path/link_to_container.tar
//
// Therefore, this package will only implement the auto-discovery and
// redirect to specific implementations once the disovery is completed
func (k *dockerResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	// 0. check if we have a tar as input
	//    detect if the tar is a container image format -> container image
	//    or a container snapshot format -> container snapshot
	// 1. check if we have a container id
	//    check if the container is running -> docker engine
	//    check if the container is stopped -> container snapshot
	// 3. check if we have an image id -> container image
	// 4. check if we have a descriptor for a registry -> container image

	uri := in.Connection

	if !strings.HasPrefix(uri, "docker://") {
		return nil, errors.New("could not find the container reference")
	}

	t := &transports.TransportConfig{}
	t.Host = strings.Replace(uri, "docker://", "", 1)

	if t == nil || len(t.Host) == 0 {
		return nil, errors.New("no endpoint provided")
	}
	log.Debug().Str("docker", t.Host).Msg("try to resolve the container or image source")

	var resolvedAsset *asset.Asset
	var err error

	// check if we are pointing to a local tar file
	_, err = os.Stat(t.Host)
	if err == nil {
		log.Debug().Msg("detected local container tar file")
		// Tar container can be an image or a snapshot
		t.Backend = transports.TransportBackend_CONNECTION_CONTAINER_TAR
		resolvedAsset = &asset.Asset{
			Connections: []*transports.TransportConfig{t},
			Platform: &platform.Platform{
				// TODO: this is temporary, decide if we want to move the detection logic for image/container here
				Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
				Runtime: transports.RUNTIME_DOCKER_IMAGE,
			},
		}
		return []*asset.Asset{resolvedAsset}, nil
	}

	log.Debug().Msg("try to connect to docker engine")
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err == nil {
		_, err := ded.ContainerInfo(t.Host)
		if err == nil {
			t.Backend = transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER
			resolvedAsset = &asset.Asset{
				Connections: []*transports.TransportConfig{t},
				Platform: &platform.Platform{
					Kind:    transports.Kind_KIND_CONTAINER,
					Runtime: transports.RUNTIME_DOCKER_CONTAINER,
				},
			}
			return []*asset.Asset{resolvedAsset}, nil
		}

		_, err = ded.ImageInfo(t.Host)
		if err == nil {
			t.Backend = transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE
			resolvedAsset = &asset.Asset{
				Connections: []*transports.TransportConfig{t},
				Platform: &platform.Platform{
					Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
					Runtime: transports.RUNTIME_DOCKER_IMAGE,
				},
			}
			return []*asset.Asset{resolvedAsset}, nil
		}
	}

	log.Debug().Msg("try to download the image from docker registry")
	_, err = name.ParseReference(t.Host, name.WeakValidation)
	if err == nil {
		t.Backend = transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY
		resolvedAsset = &asset.Asset{
			Connections: []*transports.TransportConfig{t},
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
				Runtime: transports.RUNTIME_DOCKER_REGISTRY,
			},
		}
		return []*asset.Asset{resolvedAsset}, nil
	}

	// if we reached here, we assume we have a name of an image or container from a registry
	return nil, errors.New("could not find the container reference")
}

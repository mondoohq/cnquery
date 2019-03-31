package docker

import (
	"errors"

	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/docker/docker_engine"
	"go.mondoo.io/mondoo/motor/docker/image"
	"go.mondoo.io/mondoo/motor/docker/snapshot"
	"go.mondoo.io/mondoo/motor/types"
)

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
func New(endpoint *types.Endpoint) (types.Transport, error) {

	// 0. check if we have a tar as input
	//    detect if the tar is a container image format -> container image
	//    or a container snapshot format -> container snapshot
	// 1. check if we have a container id
	//    check if the container is running -> docker engine
	//    check if the container is stopped -> container snapshot
	// 3. check if we have an image id -> container image
	// 4. check if we have a descriptor for a registry -> container image

	// TODO: check if we are pointing to a local file
	localpath := endpoint.Host + endpoint.Path
	_, err := os.Stat(localpath)
	if err == nil {
		log.Debug().Msg("found local docker/image file")
		_, err := tarball.ImageFromPath(localpath, nil)
		if err == nil {
			log.Debug().Msg("detected docker image")
			return image.NewFromFile(localpath)
		} else {
			log.Debug().Msg("detected docker container snapshot")
			return snapshot.NewFromDirectory(localpath)
		}

		// TODO: detect file format
		return nil, errors.New("could not find the container reference")
	}

	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded := NewDockerEngineDiscovery()
	if ded.IsRunning() && len(endpoint.Host) > 0 && len(endpoint.Path) == 0 {
		ci, err := ded.ContainerInfo(endpoint.Host)
		if err == nil {
			if ci.Running {
				log.Debug().Msg("found running container " + ci.ID)
				return docker_engine.New(ci.ID)
			} else {
				log.Debug().Msg("found stopped container " + ci.ID)
				return snapshot.NewFromDockerEngine(ci.ID)
			}
		}

		ii, err := ded.ImageInfo(endpoint.Host)
		if err == nil {
			log.Debug().Msg("found docker engine image " + ii.ID)
			rc, err := image.LoadFromDockerEngine(ii.ID)
			if err != nil {
				return nil, err
			}
			return image.New(rc)
		}
		return nil, errors.New("not implemented yet")
	} else {
		// load container image from remote directoryload tar file into backend
		search := endpoint.Host + endpoint.Path
		tag, err := name.NewTag(search, name.WeakValidation)
		if err == nil {
			tag.TagStr()
			log.Debug().Str("tag", tag.Name()).Msg("found valid container registry reference")
			rc, err := image.LoadFromRegistry(tag)
			if err != nil {
				return nil, err
			}
			return image.New(rc)
		} else {
			log.Debug().Str("image", search).Msg("Could not detect a valid repository url")
			return nil, err
		}

		return nil, errors.New("not implemented yet")
	}

	// if we reached here, we assume we have a name of an image or container from a registry
	return nil, errors.New("could not find the container reference")
}

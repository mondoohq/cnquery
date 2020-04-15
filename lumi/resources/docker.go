package resources

import (
	"context"
	"errors"
	"os"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/motoros/local"
)

func (p *lumiDocker) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiDocker) id() (string, error) {
	return "docker", nil
}

func (p *lumiDocker) GetImages() ([]interface{}, error) {
	_, ok := p.Runtime.Motor.Transport.(*local.LocalTransport)
	if !ok {
		return nil, errors.New("docker is not support for this transport")
	}

	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	dImages, err := cl.ImageList(context.Background(), docker_types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	imgs := make([]interface{}, len(dImages))
	for i, dImg := range dImages {

		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["id"] = dImg.ID
		args["size"] = dImg.Size
		args["virtualsize"] = dImg.VirtualSize

		labels := make(map[string]interface{})
		for key := range dImg.Labels {
			labels[key] = dImg.Labels[key]
		}
		args["labels"] = labels

		tags := []interface{}{}
		for i := range dImg.RepoTags {
			tags = append(tags, dImg.RepoTags[i])
		}
		args["tags"] = tags
		e, err := newDocker_image(p.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("docker_image", dImg.ID).Msg("lumi[docker_image]> could not create docker image resource")
			continue
		}
		imgs[i] = e.(Docker_image)
	}

	return imgs, nil
}

func (p *lumiDocker) GetContainer() ([]interface{}, error) {
	_, ok := p.Runtime.Motor.Transport.(*local.LocalTransport)
	if !ok {
		return nil, errors.New("docker is not support for this transport")
	}

	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	dContainers, err := cl.ContainerList(context.Background(), docker_types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	container := make([]interface{}, len(dContainers))
	for i, dContainer := range dContainers {

		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["id"] = dContainer.ID

		args["image"] = dContainer.Image

		args["imageid"] = dContainer.ImageID
		args["command"] = dContainer.Command
		args["state"] = dContainer.State
		args["status"] = dContainer.Status

		labels := make(map[string]interface{})
		for key := range dContainer.Labels {
			labels[key] = dContainer.Labels[key]
		}
		args["labels"] = labels

		names := []interface{}{}
		for i := range dContainer.Names {
			names = append(names, dContainer.Names[i])
		}
		args["names"] = names

		e, err := newDocker_container(p.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("docker_container", dContainer.ID).Msg("lumi[docker_container]> could not create docker container resource")
			continue
		}
		container[i] = e.(Docker_container)
	}

	return container, nil
}

func (p *lumiDocker_image) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiDocker_image) id() (string, error) {
	id, _ := p.Id()
	return id, nil
}

func (p *lumiDocker_container) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiDocker_container) id() (string, error) {
	id, _ := p.Id()
	return id, nil
}

func dockerClient() (*client.Client, error) {
	// set docker api version for macos
	os.Setenv("DOCKER_API_VERSION", "1.26")
	// Start new docker container
	return client.NewEnvClient()
}

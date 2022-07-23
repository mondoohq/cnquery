package resources

import (
	"context"
	"errors"
	"os"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.mondoo.io/mondoo/motor/transports/local"
)

func (p *lumiDocker) id() (string, error) {
	return "docker", nil
}

func (p *lumiDocker) GetImages() ([]interface{}, error) {
	_, ok := p.MotorRuntime.Motor.Transport.(*local.LocalTransport)
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
		labels := make(map[string]interface{})
		for key := range dImg.Labels {
			labels[key] = dImg.Labels[key]
		}

		tags := []interface{}{}
		for i := range dImg.RepoTags {
			tags = append(tags, dImg.RepoTags[i])
		}

		lumiDockerImage, err := p.MotorRuntime.CreateResource("docker.image",
			"id", dImg.ID,
			"size", dImg.Size,
			"virtualsize", dImg.VirtualSize,
			"labels", labels,
			"tags", tags,
		)
		if err != nil {
			return nil, err
		}

		imgs[i] = lumiDockerImage.(DockerImage)
	}

	return imgs, nil
}

func (p *lumiDocker) GetContainers() ([]interface{}, error) {
	_, ok := p.MotorRuntime.Motor.Transport.(*local.LocalTransport)
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
		labels := make(map[string]interface{})
		for key := range dContainer.Labels {
			labels[key] = dContainer.Labels[key]
		}

		names := []interface{}{}
		for i := range dContainer.Names {
			names = append(names, dContainer.Names[i])
		}

		lumiDockerContainer, err := p.MotorRuntime.CreateResource("docker.container",
			"id", dContainer.ID,
			"image", dContainer.Image,
			"imageid", dContainer.ImageID,
			"command", dContainer.Command,
			"state", dContainer.State,
			"status", dContainer.Status,
			"labels", labels,
			"names", names,
		)
		if err != nil {
			return nil, err
		}

		container[i] = lumiDockerContainer.(DockerContainer)
	}

	return container, nil
}

func (p *lumiDockerImage) id() (string, error) {
	id, _ := p.Id()
	return id, nil
}

func (p *lumiDockerContainer) id() (string, error) {
	id, _ := p.Id()
	return id, nil
}

func dockerClient() (*client.Client, error) {
	// set docker api version for macos
	os.Setenv("DOCKER_API_VERSION", "1.26")
	// Start new docker container
	return client.NewClientWithOpts(client.FromEnv)
}

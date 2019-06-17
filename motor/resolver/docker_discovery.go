package resolver

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func dockerClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

// TODO: this implementation needs to be merged with motorcloud/docker
func NewDockerEngineDiscovery() *dockerEngineDiscovery {
	dc, err := dockerClient()

	running := true
	if err != nil {
		running = false
	}

	return &dockerEngineDiscovery{
		Client:  dc,
		Running: running,
	}
}

type dockerEngineDiscovery struct {
	Running bool
	Client  *client.Client
}

func (e *dockerEngineDiscovery) IsRunning() bool {
	return e.Running
}

func (e *dockerEngineDiscovery) ContainerList() ([]string, error) {
	dc, err := dockerClient()
	if err != nil {
		return []string{}, err
	}

	containers, err := dc.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return []string{}, err
	}

	containerShas := []string{}
	for i := range containers {
		containerShas = append(containerShas, containers[i].ID)
	}

	return containerShas, nil
}

// be aware that images are prefixed with sha256:, while containers are not
func (e *dockerEngineDiscovery) ImageList() ([]string, error) {
	dc, err := dockerClient()
	if err != nil {
		return []string{}, err
	}

	images, err := dc.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return []string{}, err
	}

	imagesShas := []string{}
	for i := range images {
		imagesShas = append(imagesShas, images[i].ID)
	}

	return imagesShas, nil
}

type ContainerInfo struct {
	ID      string
	Running bool
}

// will resolve name and id to a container id
func (e *dockerEngineDiscovery) ContainerInfo(name string) (ContainerInfo, error) {
	ci := ContainerInfo{}
	dc, err := dockerClient()
	if err != nil {
		return ci, err
	}

	cdata, err := dc.ContainerInspect(context.Background(), name)
	if err != nil {
		return ci, err
	}

	ci.ID = cdata.ID
	ci.Running = cdata.State.Running
	return ci, nil
}

type ImageInfo struct {
	ID string
}

func (e *dockerEngineDiscovery) ImageInfo(name string) (ImageInfo, error) {
	ii := ImageInfo{}
	dc, err := dockerClient()
	if err != nil {
		return ii, err
	}

	res, _, err := dc.ImageInspectWithRaw(context.Background(), name)
	if err != nil {
		return ii, err
	}

	ii.ID = res.ID
	return ii, nil
}

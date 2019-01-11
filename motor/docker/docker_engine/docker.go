package docker_engine

import (
	"context"
	"errors"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
)

func New(container string) (types.Transport, error) {
	dockerClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	// check if we are having container
	data, err := dockerClient.ContainerInspect(context.Background(), container)
	if err != nil {
		return nil, errors.New("cannot find container " + container)
	}

	if !data.State.Running {
		return nil, errors.New("container " + data.ID + " is not running")
	}

	return &DockerTransport{
		dockerClient: dockerClient,
		container:    container,
	}, nil
}

type DockerTransport struct {
	dockerClient *client.Client
	container    string
}

func (t *DockerTransport) RunCommand(command string) (*types.Command, error) {
	log.Debug().Str("command", command).Msg("docker> run command")
	c := &Command{dockerClient: t.dockerClient, Container: t.container}
	res, err := c.Exec(command)
	return res, err
}

func (t *DockerTransport) File(path string) (types.File, error) {
	log.Debug().Str("path", path).Msg("docker> fetch file")
	f := &File{dockerClient: t.dockerClient, Container: t.container, filePath: path, Transport: t}
	if !f.Exists() {
		return nil, errors.New("no such file or directory")
	}
	return f, nil
}

func (t *DockerTransport) Close() {
	t.dockerClient.Close()
}

func GetDockerClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

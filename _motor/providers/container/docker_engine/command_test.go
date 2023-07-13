//go:build debugtest
// +build debugtest

package docker_engine

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startContainer() (*client.Client, string, error) {
	// os.Setenv("DOCKER_API_VERSION", "1.26")
	// Start new docker container
	ctx := context.Background()
	var err error
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, "", err
	}

	// ensure we kill container if something went wrong during assertion
	// we can ignore errors here
	dockerClient.ContainerKill(ctx, "motor-docker-test", "SIGKILL")
	dockerClient.ContainerRemove(ctx, "motor-docker-test", docker_types.ContainerRemoveOptions{Force: true})

	imageName := "ubuntu"

	out, err := dockerClient.ImagePull(ctx, imageName, docker_types.ImagePullOptions{})
	if err != nil {
		return nil, "", err
	}
	io.Copy(os.Stdout, out)

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"/bin/bash"},
		Tty:   true,
	}, nil, nil, "motor-docker-test")
	if err != nil {
		return nil, "", err
	}

	if err := dockerClient.ContainerStart(ctx, resp.ID, docker_types.ContainerStartOptions{}); err != nil {
		return nil, "", err
	}
	return dockerClient, resp.ID, nil
}

func tearDownContainer(dockerClient *client.Client, containerID string) error {
	// Stop Container
	return dockerClient.ContainerKill(context.Background(), containerID, "SIGKILL")
}

func TestDockerCommand(t *testing.T) {
	dockerClient, containerID, err := startContainer()
	require.NoError(t, err)
	defer tearDownContainer(dockerClient, containerID)

	// Execute tests
	t.Run("echo", func(t *testing.T) {
		c := &Command{dockerClient: dockerClient, Container: containerID}
		cmd, err := c.Exec("echo 'test'")
		require.NoError(t, err)
		assert.Equal(t, "echo 'test'", cmd.Command, "they should be equal")
		assert.Equal(t, nil, err, "should execute without error")

		stdout, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "test\n", string(stdout), "output should be correct")
		stderr, _ := ioutil.ReadAll(cmd.Stderr)
		assert.Equal(t, "", string(stderr), "output should be correct")
	})

	t.Run("echo pipe", func(t *testing.T) {
		cErr := &Command{dockerClient: dockerClient, Container: containerID}

		cmd, err := cErr.Exec("echo 'This message goes to stderr' >&2")
		require.NoError(t, err)

		assert.Equal(t, "echo 'This message goes to stderr' >&2", cmd.Command, "they should be equal")
		assert.Equal(t, nil, err, "should execute without error")

		stdout, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "", string(stdout), "output should be correct")

		stderr, _ := ioutil.ReadAll(cmd.Stderr)
		assert.Equal(t, "This message goes to stderr\n", string(stderr), "output should be correct")
	})
}

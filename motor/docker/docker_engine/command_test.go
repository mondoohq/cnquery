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
	"github.com/stretchr/testify/suite"
)

type DockerCommandTestSuite struct {
	suite.Suite
	ContainerID  string
	dockerClient *client.Client
}

func (suite *DockerCommandTestSuite) SetupSuite() {
	os.Setenv("DOCKER_API_VERSION", "1.26")
	// Start new docker container
	ctx := context.Background()
	var err error
	suite.dockerClient, err = client.NewEnvClient()
	if err != nil {
		suite.T().Error(err)
	}

	// ensure we kill container if something went wrong during assertion
	// we can ignore errors here
	suite.dockerClient.ContainerKill(ctx, "motor-docker-test", "SIGKILL")
	suite.dockerClient.ContainerRemove(ctx, "motor-docker-test", docker_types.ContainerRemoveOptions{Force: true})

	imageName := "ubuntu"

	out, err := suite.dockerClient.ImagePull(ctx, imageName, docker_types.ImagePullOptions{})
	if err != nil {
		suite.T().Error(err)
	}
	io.Copy(os.Stdout, out)

	resp, err := suite.dockerClient.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"/bin/bash"},
		Tty:   true,
	}, nil, nil, "motor-docker-test")
	if err != nil {
		suite.T().Error(err)
	}

	if err := suite.dockerClient.ContainerStart(ctx, resp.ID, docker_types.ContainerStartOptions{}); err != nil {
		suite.T().Error(err)
	}
	suite.ContainerID = resp.ID
}

func (suite *DockerCommandTestSuite) TestCommand() {
	// Execute tests
	c := &Command{dockerClient: suite.dockerClient, Container: suite.ContainerID}
	if assert.NotNil(suite.T(), c) {
		cmd, err := c.Exec("echo 'test'")
		assert.Equal(suite.T(), "echo 'test'", cmd.Command, "they should be equal")
		assert.Equal(suite.T(), nil, err, "should execute without error")

		stdout, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(suite.T(), "test\n", string(stdout), "output should be correct")
		stderr, _ := ioutil.ReadAll(cmd.Stderr)
		assert.Equal(suite.T(), "", string(stderr), "output should be correct")
	}

	cErr := &Command{dockerClient: suite.dockerClient, Container: suite.ContainerID}
	if assert.NotNil(suite.T(), c) {
		cmd, err := cErr.Exec("echo 'This message goes to stderr' >&2")
		assert.Equal(suite.T(), "echo 'This message goes to stderr' >&2", cmd.Command, "they should be equal")
		assert.Equal(suite.T(), nil, err, "should execute without error")

		stdout, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(suite.T(), "", string(stdout), "output should be correct")

		stderr, _ := ioutil.ReadAll(cmd.Stderr)
		assert.Equal(suite.T(), "This message goes to stderr\n", string(stderr), "output should be correct")
	}
}

func (suite *DockerCommandTestSuite) TearDownSuite() {
	// Stop Container
	ctx := context.Background()
	if err := suite.dockerClient.ContainerKill(ctx, suite.ContainerID, "SIGKILL"); err != nil {
		suite.T().Error(err)
	}
}

func TestDockerCommandTestSuite(t *testing.T) {
	suite.Run(t, new(DockerCommandTestSuite))
}

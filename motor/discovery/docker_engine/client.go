package docker_engine

import (
	"context"
	"os"
	"strings"

	"errors"
	dopts "github.com/docker/cli/opts"
	"github.com/docker/docker/client"
)

// parseDockerCLI is doing a small part from client.FromEnv(c)
// but it parses the DOCKER_HOST like the docker cli and not the docker go lib
// DO NOT ASK why docker maintains two implementations
func parseDockerCLIHost(c *client.Client) error {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		parsedHost, err := dopts.ParseHost(false, host)
		if err != nil {
			return err
		}

		if err := client.WithHost(parsedHost)(c); err != nil {
			return err
		}
	}
	return nil
}

func FromDockerEnv(c *client.Client) error {
	err := client.FromEnv(c)

	// we ignore the parse error since we are going to re-parse it anyway
	if err != nil && !strings.Contains(err.Error(), "unable to parse docker host") {
		return err
	}

	// The docker go client works different than the docker cli
	// therefore we mimic the approach from the docker cli to make it easier for users
	return parseDockerCLIHost(c)
}

func dockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(FromDockerEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

// TODO: this implementation needs to be merged with motorcloud/docker
func NewDockerEngineDiscovery() (*dockerEngineDiscovery, error) {
	dc, err := dockerClient()
	if err != nil {
		return nil, err
	}

	return &dockerEngineDiscovery{
		Client: dc,
	}, nil
}

type dockerEngineDiscovery struct {
	Client *client.Client
}

func (e *dockerEngineDiscovery) client() (*client.Client, error) {
	if e.Client != nil {
		return e.Client, nil
	}
	return nil, errors.New("docker client not initialized")
}

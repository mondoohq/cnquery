package resolver_test

import (
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports/resolver"
)

func resetEnv(env []string) {

}

func TestDockerEnvParsing(t *testing.T) {
	// reset env from https://golang.org/src/os/env_test.go
	defer func(origEnv []string) {
		for _, pair := range origEnv {
			i := strings.Index(pair[1:], "=") + 1
			if err := os.Setenv(pair[:i], pair[i+1:]); err != nil {
				t.Errorf("Setenv(%q, %q) failed during reset: %v", pair[:i], pair[i+1:], err)
			}
		}
	}(os.Environ())

	cli, err := client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.Nil(t, err)
	assert.Equal(t, "unix:///var/run/docker.sock", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "tcp://0.0.0.0:2375")
	cli, err = client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://0.0.0.0:2375", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	cli, err = client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.Nil(t, err)
	assert.Equal(t, "unix:///var/run/docker.sock", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "192.186.1.1")
	cli, err = client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.186.1.1:2375", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "http://192.186.1.1")
	cli, err = client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.NotNil(t, err)

	os.Setenv("DOCKER_HOST", "tcp://192.186.1.1")
	cli, err = client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.186.1.1:2375", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "tcp://192.168.59.103:2377")
	cli, err = client.NewClientWithOpts(resolver.FromDockerEnv)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.168.59.103:2377", cli.DaemonHost())

}

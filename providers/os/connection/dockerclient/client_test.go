// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package dockerclient_test

import (
	"os"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers/os/connection/dockerclient"
)

func newClientFromDockerEnv(t *testing.T) (*client.Client, error) {
	t.Helper()
	opts, err := dockerclient.FromDockerEnv()
	if err != nil {
		return nil, err
	}
	return client.New(opts...)
}

func TestDockerEnvParsing(t *testing.T) {
	// reset env from https://go.dev/src/os/env_test.go
	defer func(origEnv []string) {
		os.Clearenv()
		for _, pair := range origEnv {
			i := strings.Index(pair[1:], "=") + 1
			if err := os.Setenv(pair[:i], pair[i+1:]); err != nil {
				t.Errorf("Setenv(%q, %q) failed during reset: %v", pair[:i], pair[i+1:], err)
			}
		}
	}(os.Environ())

	// Isolate from the host's real ~/.docker so the "no DOCKER_HOST" default is
	// deterministic and doesn't pick up a machine-local docker context.
	os.Setenv("DOCKER_CONFIG", t.TempDir())
	os.Unsetenv("DOCKER_CONTEXT")
	os.Unsetenv("DOCKER_HOST")

	// No DOCKER_HOST and no configured context falls back to the moby default socket.
	cli, err := newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "unix:///var/run/docker.sock", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "tcp://0.0.0.0:2375")
	cli, err = newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://0.0.0.0:2375", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	cli, err = newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "unix:///var/run/docker.sock", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "192.186.1.1")
	cli, err = newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.186.1.1:2375", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "http://192.186.1.1")
	_, err = newClientFromDockerEnv(t)
	assert.NotNil(t, err)

	os.Setenv("DOCKER_HOST", "tcp://192.186.1.1")
	cli, err = newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.186.1.1:2375", cli.DaemonHost())

	os.Setenv("DOCKER_HOST", "tcp://192.168.59.103:2377")
	cli, err = newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.168.59.103:2377", cli.DaemonHost())
}

// A DOCKER_HOST always wins over any configured context, matching the docker CLI.
func TestDockerHostWinsOverContext(t *testing.T) {
	defer func(origEnv []string) {
		os.Clearenv()
		for _, pair := range origEnv {
			i := strings.Index(pair[1:], "=") + 1
			_ = os.Setenv(pair[:i], pair[i+1:])
		}
	}(os.Environ())

	os.Setenv("DOCKER_CONFIG", t.TempDir())
	os.Setenv("DOCKER_CONTEXT", "some-remote-context")
	os.Setenv("DOCKER_HOST", "tcp://192.168.59.103:2377")

	cli, err := newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "tcp://192.168.59.103:2377", cli.DaemonHost())
}

// An unknown/unresolvable context is non-fatal: we fall back to the default socket
// rather than error out.
func TestUnknownContextFallsBackToDefault(t *testing.T) {
	defer func(origEnv []string) {
		os.Clearenv()
		for _, pair := range origEnv {
			i := strings.Index(pair[1:], "=") + 1
			_ = os.Setenv(pair[:i], pair[i+1:])
		}
	}(os.Environ())

	os.Setenv("DOCKER_CONFIG", t.TempDir())
	os.Setenv("DOCKER_CONTEXT", "does-not-exist")
	os.Unsetenv("DOCKER_HOST")

	cli, err := newClientFromDockerEnv(t)
	assert.Nil(t, err)
	assert.Equal(t, "unix:///var/run/docker.sock", cli.DaemonHost())
}

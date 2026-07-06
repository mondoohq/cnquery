// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package docker_engine

import (
	"os"

	"github.com/cockroachdb/errors"
	dopts "github.com/docker/cli/opts"
	"github.com/moby/moby/client"
)

// FromDockerEnv builds client options from the environment, like [client.FromEnv],
// but it parses the DOCKER_HOST like the docker cli and not the docker go lib.
// DO NOT ASK why docker maintains two implementations.
//
// client.Opt can no longer be re-applied to an already-constructed *client.Client
// (it now closes over an unexported clientConfig used only during construction), so
// unlike the old client.FromEnv-then-override dance, the CLI-compatible host parsing
// is folded in as its own Opt in the returned slice instead. This intentionally
// composes the same pieces client.FromEnv does (minus its host parsing, which we
// replace below) rather than calling client.FromEnv directly: if a future moby/moby
// release adds a new env-derived option to FromEnv, this list won't pick it up
// automatically and will need a manual update.
func FromDockerEnv() ([]client.Opt, error) {
	opts := []client.Opt{
		client.WithTLSClientConfigFromEnv(),
		client.WithAPIVersionFromEnv(),
	}

	if host := os.Getenv(client.EnvOverrideHost); host != "" {
		parsedHost, err := dopts.ParseHost(false, host)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithHost(parsedHost))
	}

	return opts, nil
}

func dockerClient() (*client.Client, error) {
	opts, err := FromDockerEnv()
	if err != nil {
		return nil, err
	}
	// No explicit NegotiateAPIVersion call: the method was removed from *Client in
	// moby/moby's v29 client rewrite. API version negotiation now happens
	// automatically on the first request (WithAPIVersionNegotiation is a
	// documented no-op kept only for backward compatibility).
	return client.New(opts...)
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

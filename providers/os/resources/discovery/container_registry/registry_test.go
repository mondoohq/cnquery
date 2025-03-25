// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package container_registry

import (
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerRegistry(t *testing.T) {
	name := "index.docker.io"
	// url, err := url.Parse("//" + name)
	// require.NoError(t, err)

	// assert.True(t, url.Host != name)
	// assert.Equal(t, "index.docker.io", url.Host)

	if url, err := url.Parse("//" + name); err != nil || url.Host != name {
		t.Fatal(url.Host)
		// t.Fatal(fmt.Errorf("registries must be valid RFC 3986 URI authorities: %s", name))
	}

	// r := docker.NewContainerRegistryResolver()
	// assets, err := r.List(name)
	// require.NoError(t, err)
	// assert.True(t, len(assets) > 0)
}

func TestHarbor(t *testing.T) {
	urls := []string{
		"index.docker.io/library/alpine:latest",
		// 3.21.3
		"index.docker.io/library/alpine@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c",
	}

	for i := range urls {
		url := urls[i]
		ref, err := name.ParseReference(url, name.WeakValidation)
		require.NoError(t, err, url)
		assert.NotNil(t, ref, url)

		dri := DockerRegistryImages{}
		a, err := dri.toAsset(ref, nil)
		require.NoError(t, err, url)

		// check that we resolved it correctly and we got a specific shasum
		assert.Equal(t, "registry-image", a.Connections[0].Type)
		assert.True(t, strings.HasPrefix(a.Connections[0].Host, "index.docker.io/library/alpine"), url)
		assert.True(t, len(strings.Split(a.Connections[0].Host, "@")) == 2, url)
	}
}

package container_registry

import (
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
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
		"index.docker.io/library/centos:latest",
		"index.docker.io/library/centos@sha256:5528e8b1b1719d34604c87e11dcd1c0a20bedf46e83b5632cdeac91b8c04efc1",
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
		assert.Equal(t, transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY, a.Connections[0].Backend)
		assert.True(t, strings.HasPrefix(a.Connections[0].Host, "index.docker.io/library/centos"), url)
		assert.True(t, len(strings.Split(a.Connections[0].Host, "@")) == 2, url)
	}
}

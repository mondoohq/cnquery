package container_registry

import (
	"net/url"
	"testing"
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

	// r := docker.NewContainerRegistry()
	// assets, err := r.List(name)
	// require.NoError(t, err)
	// assert.True(t, len(assets) > 0)
}

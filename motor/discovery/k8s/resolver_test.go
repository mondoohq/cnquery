package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestManifestResolver(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.pod.yaml"

	assetList, err := resolver.Resolve(&providers.TransportConfig{
		Backend: providers.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, len(assetList))
}

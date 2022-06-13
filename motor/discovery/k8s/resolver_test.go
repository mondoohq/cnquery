package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
)

func TestManifestResolver(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../transports/k8s/resources/testdata/appsv1.pod.yaml"

	assetList, err := resolver.Resolve(&transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &transports.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(assetList))
}

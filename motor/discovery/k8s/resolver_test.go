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
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/hello-pod/uid/",
		Backend:    providers.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, len(assetList))
	assert.Equal(t, assetList[1].Platform.Name, "k8s-pod")
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, assetList[2].Platform.Runtime, "docker-registry")
	assert.Equal(t, assetList[2].Platform.Name, "docker-image")
}

func TestManifestResolverPodDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.pod.yaml"

	assetList, err := resolver.Resolve(&providers.TransportConfig{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/hello-pod/uid/",
		Backend:    providers.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"pods"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 2, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-manifest", assetList[1].Platform.Runtime)
	assert.Equal(t, "k8s-pod", assetList[1].Platform.Name)
}

func TestManifestResolverCronJobDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/batchv1.cronjob.yaml"

	assetList, err := resolver.Resolve(&providers.TransportConfig{
		PlatformId: "//platform/k8s/uid/123/namespace/mondoo-operator/cronjobs/name/mondoo-client-k8s-scan/uid/",
		Backend:    providers.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path":      manifestFile,
			"namespace": "mondoo-operator",
		},
		Discover: &providers.Discovery{
			Targets: []string{"cronjobs"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 2, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-cronjob", assetList[1].Platform.Name)
}

func TestManifestResolverWrongDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/batchv1.cronjob.yaml"

	assetList, err := resolver.Resolve(&providers.TransportConfig{
		Backend: providers.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path":      manifestFile,
			"namespace": "mondoo-operator",
		},
		Discover: &providers.Discovery{
			Targets: []string{"pods"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equalf(t, 1, len(assetList), "discovering pods in a cronjob manifest should only result in the manifest")
}

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestManifestResolver(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.pod.yaml"

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/hello-pod",
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, len(assetList))
	assert.Equal(t, assetList[1].Platform.Name, "k8s-pod")
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, assetList[3].Platform.Runtime, "docker-registry")
	assert.Equal(t, assetList[3].Platform.Name, "docker-image")
}

func TestManifestResolverPodDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.pod.yaml"

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/hello-pod",
		Backend:    providers.ProviderType_K8S,
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
	assert.Equal(t, 3, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-manifest", assetList[1].Platform.Runtime)
	assert.Equal(t, "k8s-pod", assetList[1].Platform.Name)
	assert.Equal(t, "default/hello-pod", assetList[1].Name)
	assert.Equal(t, "k8s-manifest", assetList[2].Platform.Runtime)
	assert.Equal(t, "k8s-pod", assetList[2].Platform.Name)
	assert.Equal(t, "default/hello-pod-2", assetList[2].Name)
}

func TestManifestResolverCronJobDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/batchv1.cronjob.yaml"

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/mondoo-operator/cronjobs/name/mondoo-client-k8s-scan",
		Backend:    providers.ProviderType_K8S,
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

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		Backend: providers.ProviderType_K8S,
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

func TestManifestResolverStatefulSetDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.statefulset.yaml"

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/statefulsets/name/mondoo-statefulset",
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"statefulsets"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 2, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-statefulset", assetList[1].Platform.Name)
}

func TestManifestResolverJobDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/batchv1.job.yaml"

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/mondoo-operator/jobs/name/mondoo-client-k8s-scan",
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path":      manifestFile,
			"namespace": "mondoo-operator",
		},
		Discover: &providers.Discovery{
			Targets: []string{"jobs"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 2, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-job", assetList[1].Platform.Name)
}

func TestManifestResolverReplicaSetDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.replicaset.yaml"

	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/replicasets/name/mondoo-replicaset",
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"replicasets"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 2, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-replicaset", assetList[1].Platform.Name)
}

func TestManifestResolverDaemonSetDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/appsv1.daemonset.yaml"

	platformId := "//platform/k8s/uid/123/namespace/default/daemonsets/name/mondoo-daemonset"
	assetList, err := resolver.Resolve(&asset.Asset{}, &providers.Config{
		PlatformId: platformId,
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"daemonsets"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 2, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-daemonset", assetList[1].Platform.Name)
	assert.Equal(t, platformId, assetList[1].Connections[0].PlatformId)
}

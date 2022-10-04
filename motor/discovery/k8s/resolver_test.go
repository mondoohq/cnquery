package k8s

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
)

func TestManifestResolver(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/pod.yaml"

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/mondoo",
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
	assert.Equal(t, assetList[3].Platform.Runtime, "docker-image")
}

func TestAdmissionReviewResolver(t *testing.T) {
	resolver := &Resolver{}

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())
	data, err := os.ReadFile("../../providers/k8s/resources/testdata/admission-review.json")
	require.NoError(t, err)

	admission := base64.StdEncoding.EncodeToString(data)
	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			k8s.OPTION_ADMISSION: admission,
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
	assert.Equal(t, assetList[2].Platform.Runtime, "docker-image")
	assert.Equal(t, assetList[3].Platform.Runtime, "k8s-admission")
}

func TestManifestResolverDiscoveries(t *testing.T) {
	testCases := []struct {
		kind            string
		discoveryOption string
		platformName    string
		numAssets       int
	}{
		{
			kind:            "pod",
			discoveryOption: "pods",
			platformName:    "k8s-pod",
			numAssets:       3,
		},
		{
			kind:            "cronjob",
			discoveryOption: "cronjobs",
			platformName:    "k8s-cronjob",
			numAssets:       2,
		},
		{
			kind:            "job",
			discoveryOption: "jobs",
			platformName:    "k8s-job",
			numAssets:       2,
		},
		{
			kind:            "statefulset",
			discoveryOption: "statefulsets",
			platformName:    "k8s-statefulset",
			numAssets:       2,
		},
		{
			kind:            "daemonset",
			discoveryOption: "daemonsets",
			platformName:    "k8s-daemonset",
			numAssets:       2,
		},
		{
			kind:            "replicaset",
			discoveryOption: "replicasets",
			platformName:    "k8s-replicaset",
			numAssets:       2,
		},
		{
			kind:            "deployment",
			discoveryOption: "deployments",
			platformName:    "k8s-deployment",
			numAssets:       2,
		},
	}

	for _, testCase := range testCases {
		t.Run("discover k8s "+testCase.kind, func(t *testing.T) {
			resolver := &Resolver{}
			manifestFile := "../../providers/k8s/resources/testdata/" + testCase.kind + ".yaml"

			ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

			assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
				PlatformId: "//platform/k8s/uid/123/namespace/default/" + testCase.discoveryOption + "/name/mondoo",
				Backend:    providers.ProviderType_K8S,
				Options: map[string]string{
					"path": manifestFile,
				},
				Discover: &providers.Discovery{
					Targets: []string{testCase.discoveryOption},
				},
			}, nil, nil)
			require.NoError(t, err)
			// When this check fails locally, check your kubeconfig.
			// context has to reference the default namespace
			assert.Equal(t, testCase.numAssets, len(assetList))
			assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
			assert.Contains(t, assetList[1].Platform.Family, "k8s")
			assert.Equal(t, "k8s-manifest", assetList[1].Platform.Runtime)
			assert.Equal(t, testCase.platformName, assetList[1].Platform.Name)
			assert.Equal(t, "default/mondoo", assetList[1].Name)
		})
	}
}

func TestManifestResolverMultiPodDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/pod.yaml"

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/mondoo",
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
	assert.Equal(t, "default/mondoo", assetList[1].Name)
	assert.Equal(t, "k8s-manifest", assetList[2].Platform.Runtime)
	assert.Equal(t, "k8s-pod", assetList[2].Platform.Name)
	assert.Equal(t, "default/hello-pod-2", assetList[2].Name)
}

func TestManifestResolverWrongDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/cronjob.yaml"

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path":      manifestFile,
			"namespace": "default",
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

func TestResourceFilter(t *testing.T) {
	cfg := &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"k8s-resources": "pod:default:nginx, pod:default:redis, deployment:test:redis, node:node1",
		},
	}

	resFilters, err := resourceFilters(cfg)
	require.NoError(t, err)

	expected := map[string][]K8sResourceIdentifier{
		"pod": {
			{Type: "pod", Namespace: "default", Name: "nginx"},
			{Type: "pod", Namespace: "default", Name: "redis"},
		},
		"deployment": {
			{Type: "deployment", Namespace: "test", Name: "redis"},
		},
		"node": {
			{Type: "node", Namespace: "", Name: "node1"},
		},
	}

	assert.Equal(t, expected, resFilters)
}

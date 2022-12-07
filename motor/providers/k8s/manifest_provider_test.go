package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
)

type K8sObjectKindTest struct {
	kind string
}

func TestManifestFiles(t *testing.T) {
	tests := []K8sObjectKindTest{
		{kind: "cronjob"},
		{kind: "job"},
		{kind: "deployment"},
		{kind: "pod"},
		{kind: "statefulset"},
		{kind: "replicaset"},
		{kind: "daemonset"},
	}
	for _, testCase := range tests {
		t.Run("k8s "+testCase.kind, func(t *testing.T) {
			manifestFile := "./resources/testdata/" + testCase.kind + ".yaml"
			transport, err := newManifestProvider("", testCase.kind, WithManifestFile(manifestFile))
			require.NoError(t, err)
			require.NotNil(t, transport)
			res, err := transport.Resources(testCase.kind, "mondoo", "default")
			require.NoError(t, err)
			assert.Equal(t, "mondoo", res.Name)
			assert.Equal(t, testCase.kind, res.Kind)
			assert.Equal(t, "k8s-manifest", transport.PlatformInfo().Runtime)
			assert.Equal(t, 1, len(res.Resources))
			podSpec, err := resources.GetPodSpec(res.Resources[0])
			require.NoError(t, err)
			assert.NotNil(t, podSpec)
			containers, err := resources.GetContainers(res.Resources[0])
			require.NoError(t, err)
			assert.Equal(t, 1, len(containers))
			initContainers, err := resources.GetInitContainers(res.Resources[0])
			require.NoError(t, err)
			assert.Equal(t, 0, len(initContainers))
		})
	}
}

func TestManifestFileProvider(t *testing.T) {
	t.Run("k8s manifest provider", func(t *testing.T) {
		manifestFile := "./resources/testdata/pod.yaml"
		transport, err := newManifestProvider("", "", WithManifestFile(manifestFile))
		require.NoError(t, err)
		require.NotNil(t, transport)
		assert.Equal(t, "k8s-manifest", transport.PlatformInfo().Name)
		assert.Equal(t, "k8s-manifest", transport.PlatformInfo().Runtime)
		assert.Equal(t, providers.Kind_KIND_CODE, transport.PlatformInfo().Kind)
		assert.Contains(t, transport.PlatformInfo().Family, "k8s")
	})
}

func TestLoadManifestDirRecursively(t *testing.T) {
	manifests, err := loadManifestFile("./resources/testdata/")
	require.NoError(t, err)

	manifestsAsString := string(manifests[:])
	// This is content from files of the root dir
	assert.Contains(t, manifestsAsString, "mondoo")
	assert.Contains(t, manifestsAsString, "RollingUpdate")

	// Files containing this should be skipped
	assert.NotContains(t, manifestsAsString, "AdmissionReview")
	assert.NotContains(t, manifestsAsString, "README")
	assert.NotContains(t, manifestsAsString, "operators.coreos.com")

	// This is from files in subdirs whicch should be included
	assert.Contains(t, manifestsAsString, "hello-1")
	assert.Contains(t, manifestsAsString, "hello-2")
	assert.Contains(t, manifestsAsString, "MondooAuditConfig")
}

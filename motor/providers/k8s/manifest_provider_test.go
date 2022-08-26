package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			transport := newManifestProvider("", WithManifestFile(manifestFile))
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

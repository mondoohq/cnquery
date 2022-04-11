package k8s

import (
	"bytes"
	"os"
	"testing"

	"go.mondoo.io/mondoo/motor/transports/k8s/resources"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestFileLoad(t *testing.T) {
	manifestFile := "./resources/testdata/appsv1.deployment.yaml"
	connector := NewManifestConnector(WithManifestFile(manifestFile))
	require.NotNil(t, connector)
	res, err := connector.Resources("deployment", "centos")
	require.NoError(t, err)
	assert.Equal(t, "centos", res.Name)
	assert.Equal(t, "deployment", res.Kind)
	assert.Equal(t, 1, len(res.RootResources))
}

func TestInmemory(t *testing.T) {
	manifestFile := "./resources/testdata/appsv1.deployment.yaml"
	data, err := os.ReadFile(manifestFile)
	require.NoError(t, err)

	objects, err := resources.ResourcesFromManifest(bytes.NewReader(data))
	require.NoError(t, err)

	connector := NewManifestConnector(WithRuntimeObjects(objects))
	require.NotNil(t, connector)
	res, err := connector.Resources("deployment", "centos")
	require.NoError(t, err)
	assert.Equal(t, "centos", res.Name)
	assert.Equal(t, "deployment", res.Kind)
	assert.Equal(t, 1, len(res.RootResources))
}

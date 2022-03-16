package resources_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	appsv1 "k8s.io/api/apps/v1"
	coresv1 "k8s.io/api/core/v1"
)

func TestLoadManifestFile(t *testing.T) {
	f, err := os.Open("./testdata/appsv1.deployment.yaml")
	if err != nil {
		t.Fatal(err)
	}

	list, err := resources.ResourcesFromManifest(f)
	require.NoError(t, err)
	assert.Equal(t, 1, len(list))

	resource := list[0]
	deployment := resource.(*appsv1.Deployment)
	assert.Equal(t, "centos", deployment.Name)
}

func TestLoadManifestDir(t *testing.T) {
	input, err := resources.MergeManifestFiles([]string{"./testdata/appsv1.deployment.yaml", "./testdata/configmap.yaml"})
	require.NoError(t, err)

	list, err := resources.ResourcesFromManifest(input)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))

	resource := list[0]
	deployment := resource.(*appsv1.Deployment)
	assert.Equal(t, "centos", deployment.Name)

	resource = list[1]
	configmap := resource.(*coresv1.ConfigMap)
	assert.Equal(t, "mondoo-daemonset-config", configmap.Name)
}

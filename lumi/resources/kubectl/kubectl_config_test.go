package kubectl_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/kubectl"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestKubectlConfigParser(t *testing.T) {
	r, err := os.Open("./testdata/kubeconfig_default.yml")
	if err != nil {
		t.Fatal(err)
	}

	config, err := kubectl.ParseKubectlConfig(r)
	require.NoError(t, err)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "minikube", config.CurrentContext)
	assert.Equal(t, "default", config.CurrentNamespace())

	r, err = os.Open("./testdata/kubeconfig_with_namespace.yml")
	if err != nil {
		t.Fatal(err)
	}

	config, err = kubectl.ParseKubectlConfig(r)
	require.NoError(t, err)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "minikube", config.CurrentContext)
	assert.Equal(t, "ggckad-s2", config.CurrentNamespace())
}

func TestKubectlExecuter(t *testing.T) {
	mock, err := mock.NewFromToml(&motorapi.Endpoint{Backend: "mock", Path: "./testdata/linux_kubeclt.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(mock)
	require.NoError(t, err)

	config, err := kubectl.LoadKubeConfig(m)
	require.NoError(t, err)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "minikube", config.CurrentContext)
}

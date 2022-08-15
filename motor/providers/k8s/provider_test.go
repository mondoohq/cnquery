//go:build debugtest
// +build debugtest

package k8s

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
)

func TestKubernetes(t *testing.T) {
	os.Setenv("DEBUG", "1")
	p, err := New(&providers.Config{
		Type: providers.ProviderType_K8S,
	})
	require.NoError(t, err)

	id, err := p.Identifier()
	require.NoError(t, err)
	fmt.Println(id)

	res, err := p.Resources("daemonsets", "")
	require.NoError(t, err)
	fmt.Println(res)

	name, err := p.Name()
	require.NoError(t, err)
	assert.Equal(t, "minikube", name)
}

func TestKubernetesManifest(t *testing.T) {
	p, err := New(&providers.Config{
		Type: providers.ProviderType_K8S,
		Options: map[string]string{
			OPTION_MANIFEST: "./resources/testdata/appsv1.daemonset.yaml",
		},
	})
	require.NoError(t, err)

	id, err := p.Identifier()
	require.NoError(t, err)
	fmt.Println(id)

	res, err := p.Resources("daemonsets", "")
	require.NoError(t, err)
	fmt.Println(res)

	name, err := p.Name()
	require.NoError(t, err)
	assert.Equal(t, "K8S Manifest testdata", name)
}

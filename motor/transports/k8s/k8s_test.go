// +build debugtest

package k8s

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
)

func TestKubernetes(t *testing.T) {
	trans, err := New(&transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_K8S,
	})
	require.NoError(t, err)

	id, err := trans.Identifier()
	require.NoError(t, err)
	fmt.Println(id)

	info, err := trans.ClusterInfo()
	require.NoError(t, err)
	assert.Equal(t, "minikube", info.Name)
}

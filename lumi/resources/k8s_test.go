package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"
)

func k8sTestQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := k8s.New(&providers.TransportConfig{
		Backend: providers.TransportBackend_CONNECTION_K8S,
		Options: map[string]string{
			"path": "./testdata/k8s",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	executor := initExecutionContext(m)
	return testQueryWithExecutor(t, executor, query, nil)
}

func TestResource_k8s(t *testing.T) {
	t.Run("k8s pod security policies", func(t *testing.T) {
		res := k8sTestQuery(t, "k8s.podSecurityPolicies[0].name")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string("example"), res[0].Data.Value)
	})
}

package os_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"
	"go.mondoo.io/mondoo/resources/packs/os"
	"go.mondoo.io/mondoo/resources/packs/testutils"
)

func k8sTestQuery(t *testing.T, query string) []*llx.RawResult {
	p, err := k8s.New(context.Background(), &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path": "./k8s/testdata",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, os.Registry)
	return x.TestQuery(t, query)
}

func TestResource_k8s(t *testing.T) {
	t.Run("k8s pod security policies", func(t *testing.T) {
		res := k8sTestQuery(t, "k8s.podSecurityPolicies[0].name")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string("example"), res[0].Data.Value)
	})
}

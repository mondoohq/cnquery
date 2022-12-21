package k8s_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	providers_k8s "go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/resources"
	core_pack "go.mondoo.com/cnquery/resources/packs/core"
	k8s_pack "go.mondoo.com/cnquery/resources/packs/k8s"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

var combinedRegistry *resources.Registry

func init() {
	combinedRegistry = k8s_pack.Registry
	combinedRegistry.Add(core_pack.Registry)
}

func Test_Ingress(t *testing.T) {
	t.Run("without-tls", func(t *testing.T) {
		p, err := providers_k8s.New(context.Background(), &providers.Config{
			Backend: providers.ProviderType_K8S,
			Options: map[string]string{
				"path": "../../../motor/providers/k8s/resources/testdata/ingress.yaml",
			},
		})
		require.NoError(t, err)

		m, err := motor.New(p)
		require.NoError(t, err)

		x := testutils.InitTester(m, combinedRegistry)
		res := x.TestQuery(t, "k8s.ingress(name: \"no-tls-ingress\", namespace: \"default\").tls.length")
		require.NotEmpty(t, res)

		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(0), res[0].Data.Value, "expected zero tls entries for Ingress without TLS data")
	})

	t.Run("with-tls", func(t *testing.T) {
		p, err := providers_k8s.New(context.Background(), &providers.Config{
			Backend: providers.ProviderType_K8S,
			Options: map[string]string{
				"path": "../../../motor/providers/k8s/resources/testdata/ingress.yaml",
			},
		})
		require.NoError(t, err)

		m, err := motor.New(p)
		require.NoError(t, err)

		x := testutils.InitTester(m, combinedRegistry)
		res := x.TestQuery(t, "k8s.ingress(name: \"ingress-with-tls\", namespace: \"default\").tls.length")
		require.NotEmpty(t, res)

		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value, "expected 1 TLS entry for test Ingress")

		res = x.TestQuery(t, "k8s.ingress(name: \"ingress-with-tls\", namespace: \"default\").tls[0].certificates[0].issuer.commonName")
		require.NotEmpty(t, res)

		assert.Equal(t, "Test Issuer", res[0].Data.Value, "unexpected value for TLS certificate issuer name")
	})
}

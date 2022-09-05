package k8s_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	k8s_pack "go.mondoo.com/cnquery/resources/packs/k8s"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

type K8sObjectKindTest struct {
	kind string
}

func k8sTestQuery(t *testing.T, query string) []*llx.RawResult {
	p, err := k8s.New(context.Background(), &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path": "../../../motor/providers/k8s/resources/testdata",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, k8s_pack.Registry)
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

func TestSupportedK8sKinds(t *testing.T) {
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
			res := k8sTestQuery(t, "k8s."+testCase.kind+"(name: \"mondoo\", namespace: \"default\"){ podSpec }")
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.NotEmpty(t, res[0].Data.Value)
		})
	}
}

func TestK8sServiceAccountAutomount(t *testing.T) {
	p, err := k8s.New(context.Background(), &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path": "../../../motor/providers/k8s/resources/testdata/serviceaccount-automount.yaml",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, k8s_pack.Registry)
	res := x.TestQuery(t, "k8s.serviceaccounts[0].automountServiceAccountToken")
	require.NotEmpty(t, res)
	assert.Empty(t, res[0].Result().Error)
	assert.Equal(t, true, res[0].Data.Value)
}

func TestK8sServiceAccountImplicitAutomount(t *testing.T) {
	p, err := k8s.New(context.Background(), &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path": "../../../motor/providers/k8s/resources/testdata/serviceaccount-implicit-automount.yaml",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, k8s_pack.Registry)
	res := x.TestQuery(t, "k8s.serviceaccounts[0].automountServiceAccountToken")
	require.NotEmpty(t, res)
	assert.Empty(t, res[0].Result().Error)
	assert.Equal(t, true, res[0].Data.Value)
}

func TestK8sServiceAccountNoAutomount(t *testing.T) {
	p, err := k8s.New(context.Background(), &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path": "../../../motor/providers/k8s/resources/testdata/serviceaccount-no-automount.yaml",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, k8s_pack.Registry)
	res := x.TestQuery(t, "k8s.serviceaccounts[0].automountServiceAccountToken")
	require.NotEmpty(t, res)
	assert.Empty(t, res[0].Result().Error)
	assert.Equal(t, false, res[0].Data.Value)
}

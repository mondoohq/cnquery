package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/tfstate"
)

func tfstateTestQuery(t *testing.T, query string) []*llx.RawResult {
	trans, err := tfstate.New(&providers.Config{
		Backend: providers.ProviderType_TERRAFORM_STATE,
		Options: map[string]string{
			"path": "./testdata/tfstate/state_simple.json",
		},
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	executor := initExecutionContext(m)
	return testQueryWithExecutor(t, executor, query, nil)
}

func TestResource_Tfstate(t *testing.T) {
	t.Run("tfstate outputs", func(t *testing.T) {
		res := tfstateTestQuery(t, "tfstate.outputs.length")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(0), res[0].Data.Value)
	})

	t.Run("tfstate recursive modules", func(t *testing.T) {
		res := tfstateTestQuery(t, "tfstate.modules.length")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("tfstate direct init", func(t *testing.T) {
		// NOTE tfstate root modules have no name
		res := tfstateTestQuery(t, `tfstate.module("").resources[0].address`)
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "aws_instance.app_server", res[0].Data.Value)
	})
}

package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/resources/packs/core/info"
	"go.mondoo.com/cnquery/resources/packs/core/kernel"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

func TestResource_KernelParameters(t *testing.T) {
	t.Run("kernel parameters", func(t *testing.T) {
		p, err := local.New()
		require.NoError(t, err)

		m, err := motor.New(p)
		require.NoError(t, err)

		tester := testutils.InitTester(m, info.Registry)

		mm, err := kernel.ResolveManager(m)
		require.NotNil(t, mm)
		require.NoError(t, err)

		res := tester.TestQuery(t, "kernel.parameters")
		assert.NotEmpty(t, res)

		params, ok := res[0].Data.Value.(map[string]interface{})
		require.True(t, ok)
		mapS := make(map[string]string)
		for k, v := range params {
			mapS[k] = v.(string)
		}

		assert.NotEmpty(t, mapS)
	})
}

func TestResource_KernelModules(t *testing.T) {
	t.Run("kernel modules", func(t *testing.T) {
		res := x.TestQuery(t, "kernel.modules")
		assert.NotEmpty(t, res)
	})

	t.Run("grab one kernel module", func(t *testing.T) {
		res := x.TestQuery(t, "kernel.modules[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "xfrm_user", res[0].Data.Value)
	})

	t.Run("grab a kernel module by name", func(t *testing.T) {
		res := x.TestQuery(t, "kernel.module('xfrm_user').size")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "36864", res[0].Data.Value)
	})
}

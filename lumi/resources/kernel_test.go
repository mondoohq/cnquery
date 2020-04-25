package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_KernelParameters(t *testing.T) {
	t.Run("kernel parameters", func(t *testing.T) {
		res := testQuery(t, "kernel.parameters")
		assert.NotEmpty(t, res)
	})

	// TODO: something is wrong with /proc parser, once fixed we need to activate this test
	// t.Run("test a specific kernel parameters", func(t *testing.T) {
	// 	res := testQuery(t, "kernel.parameters[\"net.ipv4.ip_forward\"]")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, "1", res[0].Data.Value)
	// })
}

func TestResource_KernelModules(t *testing.T) {
	t.Run("kernel modules", func(t *testing.T) {
		res := testQuery(t, "kernel.modules")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific kernel module's name", func(t *testing.T) {
		res := testQuery(t, "kernel.modules[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "xfrm_user", res[0].Data.Value)
	})
}

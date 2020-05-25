package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
)

func TestResource_Processes(t *testing.T) {
	t.Run("list processes", func(t *testing.T) {
		res := testQuery(t, "processes.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific process entry", func(t *testing.T) {
		res := testQuery(t, "processes.list[0].pid")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("test a specific process entry with filter", func(t *testing.T) {
		res := testQuery(t, "processes{ pid command }.list[0]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, map[string]interface{}{
			"1KDb20yLX9QLH98+C+vtkIBAK/8ABRHY9VVMxQ9p8Kk/c0/fJtFNgyxeQV2Na6A2C3QN4zu3ZcNN563zUwINKw==": llx.StringData("/sbin/init"),
			"8dN2acsl4BqSCpQ0kwkn+ynk/JQLd8M7XTrexlWX08tUsuTjpglEizl5RGipKJSOZDb1F+L1Otpct3c/5k7taw==": llx.IntData(1),
		}, res[0].Data.Value)
	})
}

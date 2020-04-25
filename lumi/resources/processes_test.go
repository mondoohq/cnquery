package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	// TODO: this panics the runtime
	// t.Run("test a specific process entry with filter", func(t *testing.T) {
	// 	res := testQuery(t, "processes{ pid command }.list[0]")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, map[string]interface{}{}, res[0].Data.Value)
	// })
}

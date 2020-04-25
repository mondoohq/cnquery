package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Groups(t *testing.T) {
	t.Run("list groups", func(t *testing.T) {
		res := testQuery(t, "groups.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific grroup", func(t *testing.T) {
		res := testQuery(t, "groups.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "root", res[0].Data.Value)
	})

	// TODO: this test produces a panic in llx
	// t.Run("fetch all group names", func(t *testing.T) {
	// 	res := testQuery(t, "groups.list { name }")
	// 	assert.NotEmpty(t, res)
	// 	assert.Empty(t, res[0].Result().Error)
	// 	assert.Equal(t, "root", res[0].Data.Value)
	// })
}

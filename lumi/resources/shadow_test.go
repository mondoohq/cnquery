package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Shadow(t *testing.T) {
	t.Run("list shadow entries", func(t *testing.T) {
		res := testQuery(t, "shadow.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific shadow entry", func(t *testing.T) {
		res := testQuery(t, "shadow.list[0].user")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "root", res[0].Data.Value)
	})
}

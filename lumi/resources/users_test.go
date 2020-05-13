package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Users(t *testing.T) {
	t.Run("users list", func(t *testing.T) {
		res := testQuery(t, "users.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific user's name", func(t *testing.T) {
		res := testQuery(t, "users.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "root", res[0].Data.Value)
	})

	t.Run("test contains", func(t *testing.T) {
		res := testQuery(t, "users.contains(name == 'root')")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})
}

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Platform(t *testing.T) {
	t.Run("platform info", func(t *testing.T) {
		res := testQuery(t, "platform")
		assert.NotEmpty(t, res)
	})

	t.Run("platform name", func(t *testing.T) {
		res := testQuery(t, "platform.name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "arch", res[0].Data.Value)
	})
}

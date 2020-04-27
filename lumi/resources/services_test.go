package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Services(t *testing.T) {
	t.Run("list services", func(t *testing.T) {
		res := testQuery(t, "services.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific service entry", func(t *testing.T) {
		res := testQuery(t, "services.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "pacman-init", res[0].Data.Value)
	})
}

func TestResource_Service(t *testing.T) {
	t.Run("test service", func(t *testing.T) {
		res := testQuery(t, "services.list")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific service entry", func(t *testing.T) {
		res := testQuery(t, "services.list[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "pacman-init", res[0].Data.Value)
	})
}

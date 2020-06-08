package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Mount(t *testing.T) {
	t.Run("list mount points", func(t *testing.T) {
		res := testQuery(t, "mount.list")
		assert.NotEmpty(t, res)
	})

	t.Run("check first mount entry", func(t *testing.T) {
		res := testQuery(t, "mount.list[0].device")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "overlay", res[0].Data.Value)
	})

	t.Run("search for mountpoint on root /", func(t *testing.T) {
		res := testQuery(t, "mount.where(path == \"/\").list[0].device")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "overlay", res[0].Data.Value)
	})
}

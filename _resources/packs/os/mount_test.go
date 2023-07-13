package os_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Mount(t *testing.T) {
	t.Run("list mount points", func(t *testing.T) {
		res := x.TestQuery(t, "mount.list")
		assert.NotEmpty(t, res)
	})

	t.Run("check first mount entry", func(t *testing.T) {
		res := x.TestQuery(t, "mount.list[0].device")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "overlay", res[0].Data.Value)
	})

	t.Run("search for mountpoint on root /", func(t *testing.T) {
		res := x.TestQuery(t, "mount.where(path == \"/\").list[0].device")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "overlay", res[0].Data.Value)
	})

	t.Run("check mount point resource", func(t *testing.T) {
		res := x.TestQuery(t, "mount.point(\"/dev\").mounted")
		assert.NotEmpty(t, res)
		assert.Equal(t, true, res[0].Data.Value)

		res = x.TestQuery(t, "mount.point(\"/notthere\").mounted")
		assert.NotEmpty(t, res)
		assert.Equal(t, false, res[0].Data.Value)
	})
}

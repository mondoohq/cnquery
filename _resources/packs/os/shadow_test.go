package os_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Shadow(t *testing.T) {
	t.Run("list shadow entries", func(t *testing.T) {
		res := x.TestQuery(t, "shadow.list")
		assert.NotEmpty(t, res)
		assert.Equal(t, 3, len(res[0].Data.Value.([]interface{})))
	})

	t.Run("test a specific shadow entry", func(t *testing.T) {
		res := x.TestQuery(t, "shadow.list[0].user")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "root", res[0].Data.Value)
	})

	t.Run("test empty dates that set upper bounds", func(t *testing.T) {
		res := x.TestQuery(t, "shadow.list[0].maxdays")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(math.MaxInt64), res[0].Data.Value)

		res = x.TestQuery(t, "shadow.list[0].inactivedays")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(math.MaxInt64), res[0].Data.Value)
	})

	t.Run("test empty dates that set lower bounds", func(t *testing.T) {
		res := x.TestQuery(t, "shadow.list[0].mindays")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(-1), res[0].Data.Value)

		res = x.TestQuery(t, "shadow.list[0].warndays")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(-1), res[0].Data.Value)
	})
}

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_LoginDefs(t *testing.T) {
	t.Run("logindefs params", func(t *testing.T) {
		res := testQuery(t, "logindefs.params")
		assert.NotEmpty(t, res)
	})

	t.Run("specific logindefs param", func(t *testing.T) {
		res := testQuery(t, "logindefs.params[\"UID_MIN\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "1000", res[0].Data.Value)
	})
}

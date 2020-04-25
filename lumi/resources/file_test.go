package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_File(t *testing.T) {
	t.Run("test a file", func(t *testing.T) {
		res := testQuery(t, "file(\"/etc/passwd\").exists")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("test a file c", func(t *testing.T) {
		res := testQuery(t, "file(\"/etc/passwd\").content")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.True(t, len(res[0].Data.Value.(string)) > 0)
	})
}

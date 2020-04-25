package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Command(t *testing.T) {
	t.Run("run a command", func(t *testing.T) {
		res := testQuery(t, "command(\"lsmod\").stdout")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.True(t, len(res[0].Data.Value.(string)) > 0)
	})
}

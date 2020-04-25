package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_SSHD(t *testing.T) {
	t.Run("sshd params", func(t *testing.T) {
		res := testQuery(t, "sshd.config.params")
		assert.NotEmpty(t, res)
	})

	t.Run("specific sshs param", func(t *testing.T) {
		res := testQuery(t, "sshd.config.params[\"UsePAM\"]")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "yes", res[0].Data.Value)
	})
}

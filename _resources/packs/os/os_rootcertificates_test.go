package os_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_OSRootCertificates(t *testing.T) {
	t.Run("list root certificates", func(t *testing.T) {
		res := x.TestQuery(t, "os.rootCertificates().length")
		assert.NotEmpty(t, res)
		assert.Equal(t, int64(1), res[0].Data.Value.(int64))
	})
}

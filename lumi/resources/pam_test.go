package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Pam(t *testing.T) {
	t.Run("with missing files", func(t *testing.T) {
		res := testQuery(t, "pam.conf.content")
		assert.NotEmpty(t, res)
		assert.Error(t, res[0].Data.Error, "returned an error")
	})
}

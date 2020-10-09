package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Auditpol(t *testing.T) {
	t.Run("list auditpol", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Credential Validation').list[0].subcategory")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Credential Validation", res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Credential Validation').list.length")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Credential Validation').list[0].inclusionsetting")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Success", res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Application Group Management').list { inclusionsetting == 'Success and Failure'}")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		r, found := res[0].Data.IsTruthy()
		assert.False(t, r)
		assert.True(t, found)
	})
}

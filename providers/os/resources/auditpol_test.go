// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

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
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Credential Validation')[0].subcategory")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Credential Validation", res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Credential Validation').length")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Credential Validation')[0].inclusionsetting")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Success", res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "auditpol.where(subcategory == 'Application Group Management') { inclusionsetting == 'Success and Failure'}")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		r, found := res[0].Data.IsTruthy()
		assert.False(t, r)
		assert.True(t, found)
	})

	// success / failure booleans derived from inclusionsetting, exercised
	// through the resource methods rather than the pure helper.
	successFailureCases := []struct {
		subcategory string // its inclusionsetting in the recording
		success     bool
		failure     bool
	}{
		{"System Integrity", true, true},            // "Success and Failure"
		{"Security State Change", true, false},      // "Success"
		{"Security System Extension", false, false}, // "No Auditing"
	}
	for _, tc := range successFailureCases {
		t.Run("success for "+tc.subcategory, func(t *testing.T) {
			res := testWindowsQuery(t, "auditpol.where(subcategory == '"+tc.subcategory+"')[0].success")
			assert.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.success, res[0].Data.Value)
		})
		t.Run("failure for "+tc.subcategory, func(t *testing.T) {
			res := testWindowsQuery(t, "auditpol.where(subcategory == '"+tc.subcategory+"')[0].failure")
			assert.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.failure, res[0].Data.Value)
		})
	}
}

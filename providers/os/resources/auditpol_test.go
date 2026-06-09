// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
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

// TestResource_AuditpolGerman exercises success/failure end-to-end against a
// recording of a German-localized `auditpol /r`, where both the subcategory
// names and the inclusion settings are localized ("Erfolg und Fehler" etc.).
// It proves the parse -> resource -> locale-table chain on non-English output
// and that the cnspec audit-policy queries — which select by locale-independent
// GUID and assert on the success/failure booleans — hold on a German system.
func TestResource_AuditpolGerman(t *testing.T) {
	abs, err := filepath.Abs("testdata/auditpol_windows_de.json")
	require.NoError(t, err)
	de := testutils.InitTester(testutils.RecordingMock(abs))

	// subcategoryguid -> expected booleans for its German inclusion setting.
	cases := []struct {
		guid             string
		setting          string
		success, failure bool
	}{
		{"0CCE9239-69AE-11D9-BED3-505054503030", "Erfolg und Fehler", true, true},
		{"0CCE922F-69AE-11D9-BED3-505054503030", "Erfolg", true, false},
		{"0CCE9234-69AE-11D9-BED3-505054503030", "Fehler", false, true},
		{"0CCE9211-69AE-11D9-BED3-505054503030", "Keine Überwachung", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.setting, func(t *testing.T) {
			res := de.TestQuery(t, "auditpol.where(subcategoryguid == '"+tc.guid+"')[0].success")
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.success, res[0].Data.Value, "success")

			res = de.TestQuery(t, "auditpol.where(subcategoryguid == '"+tc.guid+"')[0].failure")
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.failure, res[0].Data.Value, "failure")
		})
	}

	// The exact assertion patterns the cnspec mondoo-windows-security queries
	// use, verified against the German recording (props removed in cnspec#2744).
	patternCases := []struct {
		query string
		want  bool
	}{
		{"auditpol.where(subcategoryguid == '0CCE9217-69AE-11D9-BED3-505054503030').all(failure)", true},
		{"auditpol.where(subcategoryguid == '0CCE922F-69AE-11D9-BED3-505054503030').all(success)", true},
		{"auditpol.where(subcategoryguid == '0CCE9239-69AE-11D9-BED3-505054503030').all(success && failure)", true},
		{"auditpol.where(subcategoryguid == '0CCE9234-69AE-11D9-BED3-505054503030').all(failure && success == false)", true},
	}
	for _, tc := range patternCases {
		t.Run(tc.query, func(t *testing.T) {
			res := de.TestQuery(t, tc.query)
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.want, res[0].Data.Value)
		})
	}
}

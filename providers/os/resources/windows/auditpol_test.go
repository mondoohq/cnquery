// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/windows"
)

func TestParseAuditpol(t *testing.T) {
	mock, err := mock.New(0, "./testdata/auditpol.toml", nil)
	require.NoError(t, err)

	f, err := mock.RunCommand("auditpol /get /category:* /r")
	require.NoError(t, err)

	auditpol, err := windows.ParseAuditpol(f.Stdout)
	require.NoError(t, err)

	assert.Equal(t, 60, len(auditpol))

	expected := &windows.AuditpolEntry{
		MachineName:      "Test",
		PolicyTarget:     "System",
		Subcategory:      "Kernel Object",
		SubcategoryGUID:  "0CCE921F-69AE-11D9-BED3-505054503030",
		InclusionSetting: "No Auditing",
		ExclusionSetting: "",
	}
	found := findPol(auditpol, "Kernel Object")
	assert.Equal(t, expected, found)
}

func findPol(auditpol []windows.AuditpolEntry, subcategory string) *windows.AuditpolEntry {
	for i := range auditpol {
		if auditpol[i].Subcategory == subcategory {
			return &auditpol[i]
		}
	}
	return nil
}

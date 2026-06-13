// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

func TestParseAuditpol(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/auditpol.toml"))
	require.NoError(t, err)

	f, err := mock.RunCommand("auditpol /get /category:* /r")
	require.NoError(t, err)

	auditpol, err := windows.ParseAuditpol(f.Stdout)
	require.NoError(t, err)

	// 59 policy rows; the CSV header row is not an entry
	assert.Equal(t, 59, len(auditpol))

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

// When auditpol fails (e.g. non-admin shell) it prints a human-readable error
// instead of CSV. Previously the parser panicked with "index out of range [3]
// with length 1" on such output.
func TestParseAuditpol_NonCSVOutput(t *testing.T) {
	cases := map[string]string{
		"non-admin":         "The command must be run with administrator privileges\n",
		"error code":        "ERROR 0x00000057 occurred: The parameter is incorrect.\n",
		"empty":             "",
		"malformed midline": "Machine Name,Policy Target,Subcategory,Subcategory GUID,Inclusion Setting,Exclusion Setting\nunexpected diagnostic line\nTest,System,Logon,{0CCE9215-69AE-11D9-BED3-505054503030},Success,\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := windows.ParseAuditpol(strings.NewReader(in))
			require.NoError(t, err)
		})
	}
}

// The header row is six comma-separated column titles and previously parsed
// as a policy entry with subcategory "Subcategory".
func TestParseAuditpol_SkipsHeaderRow(t *testing.T) {
	in := "Machine Name,Policy Target,Subcategory,Subcategory GUID,Inclusion Setting,Exclusion Setting\n" +
		"Test,System,Logon,{0CCE9215-69AE-11D9-BED3-505054503030},Success,\n"
	auditpol, err := windows.ParseAuditpol(strings.NewReader(in))
	require.NoError(t, err)
	require.Len(t, auditpol, 1)
	assert.Equal(t, "Logon", auditpol[0].Subcategory)
}

func TestLookupAuditpolSubcategory(t *testing.T) {
	cases := []struct {
		guid     string
		name     string
		category string
	}{
		{"0CCE9215-69AE-11D9-BED3-505054503030", "Logon", "Logon/Logoff"},
		{"{0CCE922B-69AE-11D9-BED3-505054503030}", "Process Creation", "Detailed Tracking"},
		{"0cce923f-69ae-11d9-bed3-505054503030", "Credential Validation", "Account Logon"},
		{"0CCE9245-69AE-11D9-BED3-505054503030", "Removable Storage", "Object Access"},
	}
	for _, tc := range cases {
		t.Run(tc.guid, func(t *testing.T) {
			sub, ok := windows.LookupAuditpolSubcategory(tc.guid)
			require.True(t, ok)
			assert.Equal(t, tc.name, sub.Name)
			assert.Equal(t, tc.category, sub.Category)
		})
	}

	_, ok := windows.LookupAuditpolSubcategory("00000000-0000-0000-0000-000000000000")
	assert.False(t, ok)
}

// Every subcategory the auditpol recording reports must resolve through the
// well-known table, and vice versa — the table and a real system agree on
// all 59 subcategories and their English names.
func TestAuditpolSubcategoryTableMatchesSystem(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/auditpol.toml"))
	require.NoError(t, err)
	f, err := mock.RunCommand("auditpol /get /category:* /r")
	require.NoError(t, err)
	auditpol, err := windows.ParseAuditpol(f.Stdout)
	require.NoError(t, err)
	require.Len(t, auditpol, 59)

	for _, entry := range auditpol {
		sub, ok := windows.LookupAuditpolSubcategory(entry.SubcategoryGUID)
		require.True(t, ok, "GUID %s missing from table", entry.SubcategoryGUID)
		assert.Equal(t, entry.Subcategory, sub.Name, "name mismatch for %s", entry.SubcategoryGUID)
		assert.NotEmpty(t, sub.Category)
	}
}

func findPol(auditpol []windows.AuditpolEntry, subcategory string) *windows.AuditpolEntry {
	for i := range auditpol {
		if auditpol[i].Subcategory == subcategory {
			return &auditpol[i]
		}
	}
	return nil
}

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

func findPol(auditpol []windows.AuditpolEntry, subcategory string) *windows.AuditpolEntry {
	for i := range auditpol {
		if auditpol[i].Subcategory == subcategory {
			return &auditpol[i]
		}
	}
	return nil
}

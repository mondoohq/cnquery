package windows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/windows"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestParseAuditpol(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.Endpoint{Backend: "mock", Path: "./testdata/auditpol.toml"})
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

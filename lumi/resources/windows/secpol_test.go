package windows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/windows"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseSecpol(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/secpol.toml"})
	require.NoError(t, err)

	f, err := mock.RunCommand("powershell.exe -EncodedCommand CgBzAGUAYwBlAGQAaQB0ACAALwBlAHgAcABvAHIAdAAgAC8AYwBmAGcAIABvAHUAdAAuAGMAZgBnACAAIAB8ACAATwB1AHQALQBOAHUAbABsAAoAJAByAGEAdwAgAD0AIABHAGUAdAAtAEMAbwBuAHQAZQBuAHQAIABvAHUAdAAuAGMAZgBnAAoAUgBlAG0AbwB2AGUALQBJAHQAZQBtACAALgBcAG8AdQB0AC4AYwBmAGcAIAB8ACAATwB1AHQALQBOAHUAbABsAAoAVwByAGkAdABlAC0ATwB1AHQAcAB1AHQAIAAkAHIAYQB3AAoA")
	require.NoError(t, err)

	secpol, err := windows.ParseSecpol(f.Stdout)
	require.NoError(t, err)

	assert.Equal(t, "42", secpol.SystemAccess["MaximumPasswordAge"])
	assert.Equal(t, "chris", secpol.SystemAccess["NewAdministratorName"])
	assert.Equal(t, "0", secpol.EventAudit["AuditLogonEvents"])
	assert.Equal(t, []interface{}{"S-1-1-0", "S-1-5-32-544", "S-1-5-32-545", "S-1-5-32-551"}, secpol.PrivilegeRights["SeNetworkLogonRight"])
	assert.Equal(t, "3,0", secpol.RegistryValues["MACHINE\\System\\CurrentControlSet\\Control\\Lsa\\FullPrivilegeAuditing"])
}

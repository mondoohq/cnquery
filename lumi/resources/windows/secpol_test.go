package windows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestParseSecpol(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/secpol.toml")
	require.NoError(t, err)

	encoded := powershell.Encode(SecpolScript)
	f, err := mock.RunCommand(encoded)
	require.NoError(t, err)

	secpol, err := ParseSecpol(f.Stdout)
	require.NoError(t, err)

	assert.Equal(t, "42", secpol.SystemAccess["MaximumPasswordAge"])
	assert.Equal(t, "chris", secpol.SystemAccess["NewAdministratorName"])
	assert.Equal(t, "0", secpol.EventAudit["AuditLogonEvents"])
	assert.Equal(t, []interface{}{"S-1-1-0", "S-1-5-32-544", "S-1-5-32-545", "S-1-5-32-551"}, secpol.PrivilegeRights["SeNetworkLogonRight"])
	assert.Equal(t, "3,0", secpol.RegistryValues["MACHINE\\System\\CurrentControlSet\\Control\\Lsa\\FullPrivilegeAuditing"])
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

func TestParseSecpol(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/secpol.toml"))
	require.NoError(t, err)

	encoded := powershell.Encode(SecpolScript)
	f, err := mock.RunCommand(encoded)
	require.NoError(t, err)

	secpol, err := ParseSecpol(f.Stdout)
	require.NoError(t, err)

	assert.Equal(t, "42", secpol.SystemAccess["MaximumPasswordAge"])
	assert.Equal(t, "chris", secpol.SystemAccess["NewAdministratorName"])
	assert.Equal(t, "0", secpol.EventAudit["AuditLogonEvents"])
	assert.Equal(t, []any{"S-1-1-0", "S-1-5-32-544", "S-1-5-32-545", "S-1-5-32-551"}, secpol.PrivilegeRights["SeNetworkLogonRight"])
	assert.Equal(t, "3,0", secpol.RegistryValues["MACHINE\\System\\CurrentControlSet\\Control\\Lsa\\FullPrivilegeAuditing"])
}

func TestParseSecpolWithNonSIDEntries(t *testing.T) {
	// Simulate secedit output where the script could not resolve a name to a SID.
	// The parser should keep secpol.privilegerights output restricted to SIDs.
	input := `[Unicode]
Unicode=yes
[System Access]
MinimumPasswordAge = 0
[Event Audit]
AuditSystemEvents = 0
[Registry Values]
MACHINE\System\foo=4,0
[Privilege Rights]
SeDenyNetworkLogonRight = Guest,*S-1-5-32-544
SeInteractiveLogonRight = *S-1-5-32-544,*S-1-5-32-545,Gast
[Version]
signature="$CHICAGO$"
Revision=1
`
	secpol, err := ParseSecpol(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, []any{"S-1-5-32-544"}, secpol.PrivilegeRights["SeDenyNetworkLogonRight"])
	assert.Equal(t, []any{"S-1-5-32-544", "S-1-5-32-545"}, secpol.PrivilegeRights["SeInteractiveLogonRight"])
}

func TestParseSecpolNormalizesPrivilegeRightSIDs(t *testing.T) {
	input := `[Unicode]
Unicode=yes
[System Access]
MinimumPasswordAge = 0
[Event Audit]
AuditSystemEvents = 0
[Registry Values]
MACHINE\System\foo=4,0
[Privilege Rights]
SeDenyNetworkLogonRight = S-1-5-32-545,*S-1-5-32-544,Guest,,not-a-sid,S-1-X-0
[Version]
signature="$CHICAGO$"
Revision=1
`
	secpol, err := ParseSecpol(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, []any{"S-1-5-32-544", "S-1-5-32-545"}, secpol.PrivilegeRights["SeDenyNetworkLogonRight"])
}

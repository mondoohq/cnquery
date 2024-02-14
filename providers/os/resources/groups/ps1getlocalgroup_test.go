// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package groups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/groups"
)

func TestWindowsGroupsParserFromMock(t *testing.T) {
	mock, err := mock.New(0, "./testdata/windows.toml", &inventory.Asset{})
	require.NoError(t, err)

	f, err := mock.RunCommand("powershell -c \"Get-LocalGroup | ConvertTo-Json\"")
	require.NoError(t, err)

	localGroups, err := groups.ParseWindowsLocalGroups(f.Stdout)
	require.NoError(t, err)

	assert.Equal(t, 25, len(localGroups))

	expected := &groups.WindowsLocalGroup{
		Name:            "Administrators",
		Description:     "Administrators have complete and unrestricted access to the computer/domain",
		ObjectClass:     "Group",
		PrincipalSource: 1,
		SID: groups.WindowsSID{
			BinaryLength:     16,
			AccountDomainSid: nil,
			Value:            "S-1-5-32-544",
		},
	}
	found := findWindowsGroup(localGroups, "Administrators")

	assert.EqualValues(t, expected, found)
}

func findWindowsGroup(localGroups []groups.WindowsLocalGroup, name string) *groups.WindowsLocalGroup {
	for i := range localGroups {
		if localGroups[i].Name == name {
			return &localGroups[i]
		}
	}
	return nil
}

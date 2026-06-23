// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	directory "google.golang.org/api/admin/directory/v1"
)

func TestRoleAssignmentToData(t *testing.T) {
	entry := &directory.RoleAssignment{
		RoleAssignmentId: 9876543210,
		RoleId:           12345,
		AssignedTo:       "103456789012345678901",
		AssigneeType:     "user",
		ScopeType:        "ORG_UNIT",
		OrgUnitId:        "03ph8a2z1xdnme9",
		Condition:        "api.getAttribute('...')",
	}

	d := roleAssignmentToData(entry)
	// int64 ids render as decimal strings (no scientific notation / truncation)
	require.Equal(t, "9876543210", d.ID)
	require.Equal(t, int64(12345), d.RoleID)
	require.Equal(t, "103456789012345678901", d.AssignedTo)
	require.Equal(t, "user", d.AssigneeType)
	require.Equal(t, "ORG_UNIT", d.ScopeType)
	require.Equal(t, "03ph8a2z1xdnme9", d.OrgUnitID)
	require.Equal(t, "api.getAttribute('...')", d.Condition)
}

func TestRoleAssignmentToData_GroupAssignee(t *testing.T) {
	d := roleAssignmentToData(&directory.RoleAssignment{
		RoleAssignmentId: 1,
		RoleId:           2,
		AssignedTo:       "01234567890",
		AssigneeType:     "group",
		ScopeType:        "CUSTOMER",
	})
	require.Equal(t, "group", d.AssigneeType)
	require.Empty(t, d.OrgUnitID)
	require.Empty(t, d.Condition)
}

func TestIsUserAssignee(t *testing.T) {
	require.True(t, isUserAssignee("user"))
	require.False(t, isUserAssignee("group"))
	require.False(t, isUserAssignee("USER")) // API returns lowercase; guard against case drift
	require.False(t, isUserAssignee(""))
}

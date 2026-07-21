// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	cosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCosmosRoleDefinitionArgs(t *testing.T) {
	t.Run("empty definition defaults every field", func(t *testing.T) {
		args, err := cosmosRoleDefinitionArgs(cosmosRoleDefinition{})
		require.NoError(t, err)
		assert.Equal(t, "", args["roleName"].Value)
		assert.Equal(t, "", args["roleType"].Value)
		assert.Equal(t, []any{}, args["assignableScopes"].Value)
		assert.Equal(t, []any{}, args["permissions"].Value)
	})

	t.Run("populated definition maps to typed values", func(t *testing.T) {
		d := cosmosRoleDefinition{
			id:       to.Ptr("/subscriptions/s/resourceGroups/rg/providers/Microsoft.DocumentDB/databaseAccounts/a/cassandraRoleDefinitions/abc"),
			name:     to.Ptr("abc"),
			typ:      to.Ptr("Microsoft.DocumentDB/databaseAccounts/cassandraRoleDefinitions"),
			roleName: to.Ptr("My Reader"),
			roleType: to.Ptr(cosmos.RoleDefinitionTypeBuiltInRole),
			// second scope is nil and must be dropped
			scopes: []*string{to.Ptr("/scope/a"), nil, to.Ptr("/scope/b")},
			// second permission is nil and must be dropped
			permissions: []*cosmos.Permission{
				{DataActions: []*string{to.Ptr("Microsoft.DocumentDB/.../read")}},
				nil,
			},
		}
		args, err := cosmosRoleDefinitionArgs(d)
		require.NoError(t, err)

		// __id and id both derive from the ARM id
		assert.Equal(t, *d.id, args["id"].Value)
		assert.Equal(t, *d.id, args["__id"].Value)
		assert.Equal(t, "abc", args["name"].Value)
		assert.Equal(t, "My Reader", args["roleName"].Value)
		// enum coerced to its string form
		assert.Equal(t, "BuiltInRole", args["roleType"].Value)
		// nil scope dropped
		assert.Equal(t, []any{"/scope/a", "/scope/b"}, args["assignableScopes"].Value)

		// nil permission dropped, the rest converted to dicts
		perms, ok := args["permissions"].Value.([]any)
		require.True(t, ok)
		require.Len(t, perms, 1)
		perm, ok := perms[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []any{"Microsoft.DocumentDB/.../read"}, perm["dataActions"])
	})
}

func TestCosmosRoleAssignmentArgs(t *testing.T) {
	t.Run("empty assignment defaults every field", func(t *testing.T) {
		args := cosmosRoleAssignmentArgs(cosmosRoleAssignment{})
		assert.Equal(t, "", args["principalId"].Value)
		assert.Equal(t, "", args["roleDefinitionId"].Value)
		assert.Equal(t, "", args["scope"].Value)
		assert.Equal(t, "", args["provisioningState"].Value)
	})

	t.Run("populated assignment maps to typed values", func(t *testing.T) {
		ra := cosmosRoleAssignment{
			id:                to.Ptr("/subscriptions/s/resourceGroups/rg/providers/Microsoft.DocumentDB/databaseAccounts/a/cassandraRoleAssignments/xyz"),
			name:              to.Ptr("xyz"),
			typ:               to.Ptr("Microsoft.DocumentDB/databaseAccounts/cassandraRoleAssignments"),
			principalID:       to.Ptr("00000000-0000-0000-0000-000000000000"),
			roleDefinitionID:  to.Ptr("/roleDefinitions/abc"),
			scope:             to.Ptr("/subscriptions/s/.../databaseAccounts/a"),
			provisioningState: to.Ptr("Succeeded"),
		}
		args := cosmosRoleAssignmentArgs(ra)

		assert.Equal(t, *ra.id, args["id"].Value)
		assert.Equal(t, *ra.id, args["__id"].Value)
		assert.Equal(t, "xyz", args["name"].Value)
		assert.Equal(t, *ra.principalID, args["principalId"].Value)
		assert.Equal(t, *ra.roleDefinitionID, args["roleDefinitionId"].Value)
		assert.Equal(t, *ra.scope, args["scope"].Value)
		assert.Equal(t, "Succeeded", args["provisioningState"].Value)
	})
}

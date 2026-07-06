// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func roleAssignmentFor(principalID string) *mqlAzureSubscriptionAuthorizationServiceRoleAssignment {
	return &mqlAzureSubscriptionAuthorizationServiceRoleAssignment{
		PrincipalId: plugin.TValue[string]{Data: principalID, State: plugin.StateIsSet},
	}
}

func TestFilterRoleAssignmentsByPrincipal(t *testing.T) {
	assignments := []any{
		roleAssignmentFor("aaa"),
		roleAssignmentFor("bbb"),
		roleAssignmentFor("aaa"),
		roleAssignmentFor("ccc"),
	}

	t.Run("returns every assignment for the principal", func(t *testing.T) {
		got := filterRoleAssignmentsByPrincipal(assignments, "aaa")
		assert.Len(t, got, 2)
		for _, ra := range got {
			assert.Equal(t, "aaa", ra.(*mqlAzureSubscriptionAuthorizationServiceRoleAssignment).PrincipalId.Data)
		}
	})
	t.Run("single match", func(t *testing.T) {
		assert.Len(t, filterRoleAssignmentsByPrincipal(assignments, "bbb"), 1)
	})
	t.Run("no match yields empty, non-nil slice", func(t *testing.T) {
		got := filterRoleAssignmentsByPrincipal(assignments, "zzz")
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("empty input yields empty slice", func(t *testing.T) {
		assert.Empty(t, filterRoleAssignmentsByPrincipal(nil, "aaa"))
	})
	t.Run("skips entries of the wrong type", func(t *testing.T) {
		mixed := []any{roleAssignmentFor("aaa"), "not-a-role-assignment", 42}
		assert.Len(t, filterRoleAssignmentsByPrincipal(mixed, "aaa"), 1)
	})
}

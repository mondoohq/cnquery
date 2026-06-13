// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPolicyAssignment is a small helper to build a PolicyAssignment fixture
// with the fields policyAssignmentArgs reads.
func newPolicyAssignment(id, displayName, scope, policyDefinitionID, enforcementMode string) PolicyAssignment {
	pa := PolicyAssignment{ID: id}
	pa.Properties.DisplayName = displayName
	pa.Properties.Scope = scope
	pa.Properties.PolicyDefinitionID = policyDefinitionID
	pa.Properties.EnforcementMode = enforcementMode
	return pa
}

func TestPolicyAssignmentArgs(t *testing.T) {
	t.Run("subscription-scoped assignment", func(t *testing.T) {
		pa := newPolicyAssignment(
			"/subscriptions/sub-123/providers/Microsoft.Authorization/policyAssignments/direct",
			"Directly assigned policy",
			"/subscriptions/sub-123",
			"/providers/Microsoft.Authorization/policyDefinitions/def-1",
			"Default",
		)

		args, err := policyAssignmentArgs(pa)
		require.NoError(t, err)

		assert.Equal(t, "/subscriptions/sub-123/providers/Microsoft.Authorization/policyAssignments/direct", args["assignmentId"].Value)
		assert.Equal(t, "Directly assigned policy", args["name"].Value)
		assert.Equal(t, "/subscriptions/sub-123", args["scope"].Value)
		assert.Equal(t, "/providers/Microsoft.Authorization/policyDefinitions/def-1", args["id"].Value)
		assert.Equal(t, "Default", args["enforcementMode"].Value)
	})

	// Regression guard for issue #5812: policy assignments inherited from a
	// management group that contains the subscription must be surfaced. The
	// subscription-level list returns them with a managementGroups scope, and
	// the mapping has to preserve that scope rather than drop or rewrite it.
	t.Run("management-group inherited assignment", func(t *testing.T) {
		pa := newPolicyAssignment(
			"/providers/Microsoft.Management/managementGroups/mg-root/providers/Microsoft.Authorization/policyAssignments/inherited",
			"Inherited from management group",
			"/providers/Microsoft.Management/managementGroups/mg-root",
			"/providers/Microsoft.Authorization/policyDefinitions/def-2",
			"DoNotEnforce",
		)

		args, err := policyAssignmentArgs(pa)
		require.NoError(t, err)

		assert.Equal(t, "/providers/Microsoft.Management/managementGroups/mg-root/providers/Microsoft.Authorization/policyAssignments/inherited", args["assignmentId"].Value)
		assert.Equal(t, "Inherited from management group", args["name"].Value)
		// The scope stays a managementGroups path so callers can tell the
		// assignment is inherited rather than directly assigned.
		assert.Equal(t, "/providers/Microsoft.Management/managementGroups/mg-root", args["scope"].Value)
		assert.Equal(t, "DoNotEnforce", args["enforcementMode"].Value)
	})
}

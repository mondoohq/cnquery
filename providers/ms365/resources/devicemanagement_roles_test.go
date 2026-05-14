// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/stretchr/testify/assert"
)

func TestStringsToAny(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []any
	}{
		{"empty slice", []string{}, []any{}},
		{"nil slice", nil, []any{}},
		{"single value", []string{"foo"}, []any{"foo"}},
		{"multiple values", []string{"a", "b", "c"}, []any{"a", "b", "c"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, stringsToAny(tc.in))
		})
	}
}

func TestRolePermissionsToDicts_Empty(t *testing.T) {
	assert.Equal(t, []any{}, rolePermissionsToDicts(nil))
	assert.Equal(t, []any{}, rolePermissionsToDicts([]betamodels.RolePermissionable{}))
}

func TestRolePermissionsToDicts_WithResourceActions(t *testing.T) {
	action := betamodels.NewResourceAction()
	action.SetAllowedResourceActions([]string{"action.read", "action.write"})
	action.SetNotAllowedResourceActions([]string{"action.delete"})

	perm := betamodels.NewRolePermission()
	perm.SetResourceActions([]betamodels.ResourceActionable{action})

	got := rolePermissionsToDicts([]betamodels.RolePermissionable{perm})

	assert.Len(t, got, 1)
	entry, ok := got[0].(map[string]any)
	assert.True(t, ok)
	actions, ok := entry["resourceActions"].([]any)
	assert.True(t, ok)
	assert.Len(t, actions, 1)
	actionMap, ok := actions[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, []any{"action.read", "action.write"}, actionMap["allowedResourceActions"])
	assert.Equal(t, []any{"action.delete"}, actionMap["notAllowedResourceActions"])
}

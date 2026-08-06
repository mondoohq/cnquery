// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oktaRoleTypedRefs reads the custom-role and resource-set ids out of a role
// assignment's HAL `_links`. The v5 SDK types only the `self` link, so the rest
// come from an untyped map keyed by link name. Nothing fails loudly if those
// key names are wrong: the accessors just resolve to null, which is
// indistinguishable from a standard admin role that legitimately has neither.
// These tests pin the wire shape so a rename cannot silently empty the
// admin-privilege graph.
func TestOktaRoleTypedRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		wire            string
		wantCustomRole  string
		wantResourceSet string
	}{
		{
			name: "custom role scoped to a resource set",
			wire: `{
				"id": "ra1",
				"type": "CUSTOM",
				"_links": {
					"self":         {"href": "https://x.okta.com/api/v1/users/00u1/roles/ra1"},
					"permissions":  {"href": "https://x.okta.com/api/v1/iam/roles/cr1/permissions"},
					"resource-set": {"href": "https://x.okta.com/api/v1/iam/resource-sets/rs1"}
				}
			}`,
			wantCustomRole:  "cr1",
			wantResourceSet: "rs1",
		},
		{
			name: "custom role without a permissions link falls back to role",
			wire: `{
				"id": "ra2",
				"type": "CUSTOM",
				"_links": {
					"role":         {"href": "https://x.okta.com/api/v1/iam/roles/cr2"},
					"resource-set": {"href": "https://x.okta.com/api/v1/iam/resource-sets/rs2"}
				}
			}`,
			wantCustomRole:  "cr2",
			wantResourceSet: "rs2",
		},
		{
			name: "standard admin role carries neither",
			wire: `{
				"id": "ra3",
				"type": "SUPER_ADMIN",
				"_links": {"self": {"href": "https://x.okta.com/api/v1/users/00u1/roles/ra3"}}
			}`,
			wantCustomRole:  "",
			wantResourceSet: "",
		},
		{
			name: "standard role is never treated as custom even with a permissions link",
			wire: `{
				"id": "ra4",
				"type": "READ_ONLY_ADMIN",
				"_links": {"permissions": {"href": "https://x.okta.com/api/v1/iam/roles/cr4/permissions"}}
			}`,
			wantCustomRole:  "",
			wantResourceSet: "",
		},
		{
			name:            "no links at all",
			wire:            `{"id": "ra5", "type": "CUSTOM"}`,
			wantCustomRole:  "",
			wantResourceSet: "",
		},
		{
			name: "malformed link entries are ignored, not fatal",
			wire: `{
				"id": "ra6",
				"type": "CUSTOM",
				"_links": {"permissions": "not-an-object", "resource-set": {"nothref": 1}}
			}`,
			wantCustomRole:  "",
			wantResourceSet: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var role okta.Role
			require.NoError(t, json.Unmarshal([]byte(tc.wire), &role))

			customRole, resourceSet := oktaRoleTypedRefs(&role)
			assert.Equal(t, tc.wantCustomRole, customRole, "custom role id")
			assert.Equal(t, tc.wantResourceSet, resourceSet, "resource set id")
		})
	}
}

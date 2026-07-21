// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	cloudidentity "google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/googleapi"
)

func TestPolicyToData(t *testing.T) {
	t.Run("admin policy with org-unit scope and decoded value", func(t *testing.T) {
		entry := &cloudidentity.Policy{
			Name: "policies/abc123",
			Type: "ADMIN",
			PolicyQuery: &cloudidentity.PolicyQuery{
				OrgUnit: "orgUnits/03ph8a2z1xdnme9",
				Query:   "entity.org_units.exists(...)",
			},
			Setting: &cloudidentity.Setting{
				Type:  "settings/security.session_controls",
				Value: googleapi.RawMessage(`{"webSessionDuration":"36000s"}`),
			},
		}

		d := policyToData(entry)
		require.Equal(t, "policies/abc123", d.Name)
		require.Equal(t, "ADMIN", d.Type)
		require.Equal(t, "settings/security.session_controls", d.SettingType)
		require.Equal(t, "orgUnits/03ph8a2z1xdnme9", d.OrgUnit)
		require.Empty(t, d.Group)
		require.Equal(t, "entity.org_units.exists(...)", d.Query)
		require.Equal(t, map[string]any{"webSessionDuration": "36000s"}, d.Value)
	})

	t.Run("group-scoped policy", func(t *testing.T) {
		entry := &cloudidentity.Policy{
			Name:        "policies/grp",
			Type:        "ADMIN",
			PolicyQuery: &cloudidentity.PolicyQuery{Group: "groups/01234"},
			Setting:     &cloudidentity.Setting{Type: "settings/security.password"},
		}

		d := policyToData(entry)
		require.Equal(t, "groups/01234", d.Group)
		require.Empty(t, d.OrgUnit)
		require.Equal(t, "settings/security.password", d.SettingType)
		// no value payload -> nil (surfaces as null field)
		require.Nil(t, d.Value)
	})

	t.Run("nil PolicyQuery and nil Setting are tolerated", func(t *testing.T) {
		d := policyToData(&cloudidentity.Policy{Name: "policies/x", Type: "SYSTEM"})
		require.Equal(t, "policies/x", d.Name)
		require.Equal(t, "SYSTEM", d.Type)
		require.Empty(t, d.SettingType)
		require.Empty(t, d.OrgUnit)
		require.Empty(t, d.Group)
		require.Nil(t, d.Value)
	})

	t.Run("invalid JSON value leaves Value nil without erroring", func(t *testing.T) {
		entry := &cloudidentity.Policy{
			Name:    "policies/bad",
			Setting: &cloudidentity.Setting{Type: "settings/x", Value: googleapi.RawMessage(`not json`)},
		}
		d := policyToData(entry)
		require.Equal(t, "settings/x", d.SettingType)
		require.Nil(t, d.Value)
	})
}

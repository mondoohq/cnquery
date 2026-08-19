// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixture decodes a captured API response into the SDK type the resource
// reads it through, which is what pins the SDK's own field tags against the
// payload a real agent serves.
func loadFixture[T any](t *testing.T, name string, out *T) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, out))
}

func TestAclDefaultDeny(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		defaultPolicy string
		want          bool
	}{
		{"enabled and deny", true, "deny", true},
		{"enabled and allow", true, "allow", false},
		// a default policy of deny on an agent with ACLs switched off denies
		// nothing, and reporting true here would pass the check that matters
		{"disabled but deny", false, "deny", false},
		{"disabled and allow", false, "allow", false},
		{"enabled with nothing reported", true, "", false},
		{"case and padding do not change the answer", true, " Deny ", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, aclDefaultDeny(tc.enabled, tc.defaultPolicy))
		})
	}
}

func TestLinksIncludeGlobalManagement(t *testing.T) {
	t.Run("matched by reserved identifier", func(t *testing.T) {
		// renaming the built-in policy must not hide it
		assert.True(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{
			{ID: globalManagementPolicyID, Name: "renamed-by-an-operator"},
		}))
	})

	t.Run("matched by name", func(t *testing.T) {
		assert.True(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{
			{ID: "aaaaaaaa-0000-0000-0000-000000000000", Name: globalManagementPolicyName},
		}))
	})

	t.Run("no match", func(t *testing.T) {
		assert.False(t, linksIncludeGlobalManagement(nil))
		assert.False(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{}))
		assert.False(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{
			{ID: "aaaaaaaa-0000-0000-0000-000000000000", Name: "web-read"},
		}))
		// the read-only built-in is not the unrestricted one
		assert.False(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{
			{ID: "00000000-0000-0000-0000-000000000002", Name: "builtin/global-read-only"},
		}))
	})

	t.Run("nil entries are skipped rather than panicking", func(t *testing.T) {
		assert.False(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{nil}))
		assert.True(t, linksIncludeGlobalManagement([]*consulapi.ACLLink{
			nil, {Name: globalManagementPolicyName},
		}))
	})
}

func TestIsGlobalManagementPolicy(t *testing.T) {
	assert.True(t, isGlobalManagementPolicy(globalManagementPolicyID, "anything"))
	assert.True(t, isGlobalManagementPolicy("anything", globalManagementPolicyName))
	assert.False(t, isGlobalManagementPolicy("00000000-0000-0000-0000-000000000002", "builtin/global-read-only"))
	assert.False(t, isGlobalManagementPolicy("", ""))
}

func TestTokenListFixtureDecodesAndDerives(t *testing.T) {
	var entries []*consulapi.ACLTokenListEntry
	loadFixture(t, "acl-tokens.json", &entries)
	require.Len(t, entries, 3)

	byDescription := map[string]aclTokenRecord{}
	for _, entry := range entries {
		byDescription[entry.Description] = tokenRecordFromListEntry(entry)
	}

	t.Run("management token", func(t *testing.T) {
		record := byDescription["Initial Management Token"]
		assert.Equal(t, "70606984-d544-3def-9dd2-c9fa325a0b74", record.AccessorID)
		assert.False(t, record.Local)
		assert.True(t, record.hasGrants())
		assert.True(t, linksIncludeGlobalManagement(record.Policies))
		assert.Nil(t, record.ExpirationTime, "a token with no expiry never expires")
		assert.False(t, record.CreateTime.IsZero())
	})

	t.Run("anonymous token", func(t *testing.T) {
		record := byDescription["Anonymous Token"]
		assert.Equal(t, anonymousTokenAccessorID, record.AccessorID)
		// this fixture was captured after granting the anonymous token the
		// read-only built-in, which is the state a check must catch
		assert.True(t, record.hasGrants())
		assert.False(t, linksIncludeGlobalManagement(record.Policies))
		require.Len(t, record.Policies, 1)
		assert.Equal(t, "builtin/global-read-only", record.Policies[0].Name)
	})

	t.Run("expiring token", func(t *testing.T) {
		record := byDescription["app token"]
		require.NotNil(t, record.ExpirationTime)
		assert.True(t, record.ExpirationTime.After(record.CreateTime))
		require.Len(t, record.Policies, 1)
		require.Len(t, record.Roles, 1)
		assert.Equal(t, "web-role", record.Roles[0].Name)
	})
}

// The list endpoint hands the secret half of every token to a caller holding a
// management token. Nothing that reaches the schema may carry it, so the
// normalized record is swept for the fixture's secrets by reflection rather
// than by remembering to check each field.
func TestTokenRecordCarriesNoSecret(t *testing.T) {
	var entries []*consulapi.ACLTokenListEntry
	loadFixture(t, "acl-tokens.json", &entries)

	secrets := []string{}
	for _, entry := range entries {
		require.NotEmpty(t, entry.SecretID, "the fixture must actually contain a secret")
		if entry.SecretID != "anonymous" {
			secrets = append(secrets, entry.SecretID)
		}
	}
	require.NotEmpty(t, secrets)

	for _, entry := range entries {
		rendered := renderRecord(t, tokenRecordFromListEntry(entry))
		for _, secret := range secrets {
			assert.NotContains(t, rendered, secret,
				"the token secret must never reach the schema")
		}
	}

	var single consulapi.ACLToken
	loadFixture(t, "acl-token-anonymous.json", &single)
	rendered := renderRecord(t, tokenRecordFromToken(&single))
	for _, secret := range secrets {
		assert.NotContains(t, rendered, secret)
	}

	// the record type must not grow a secret-carrying field later either
	fields := reflect.TypeOf(aclTokenRecord{})
	for i := 0; i < fields.NumField(); i++ {
		name := strings.ToLower(fields.Field(i).Name)
		assert.NotContains(t, name, "secret",
			"aclTokenRecord must not carry a secret field")
	}
}

func renderRecord(t *testing.T, record aclTokenRecord) string {
	t.Helper()
	encoded, err := json.Marshal(record)
	require.NoError(t, err)
	return string(encoded)
}

func TestTokenRecordFromTokenMatchesListEntry(t *testing.T) {
	// the two endpoints return different types for the same token, and the
	// resource maps both through one record, so they must agree
	var single consulapi.ACLToken
	loadFixture(t, "acl-token-anonymous.json", &single)

	var entries []*consulapi.ACLTokenListEntry
	loadFixture(t, "acl-tokens.json", &entries)

	var listed *consulapi.ACLTokenListEntry
	for _, entry := range entries {
		if entry.AccessorID == anonymousTokenAccessorID {
			listed = entry
		}
	}
	require.NotNil(t, listed)

	fromSingle := tokenRecordFromToken(&single)
	fromList := tokenRecordFromListEntry(listed)

	assert.Equal(t, fromList.AccessorID, fromSingle.AccessorID)
	assert.Equal(t, fromList.Description, fromSingle.Description)
	assert.Equal(t, fromList.Local, fromSingle.Local)
	assert.Equal(t, len(fromList.Policies), len(fromSingle.Policies))
	assert.Equal(t, fromList.hasGrants(), fromSingle.hasGrants())
}

func TestTokenRecordFromNil(t *testing.T) {
	assert.Equal(t, aclTokenRecord{}, tokenRecordFromListEntry(nil))
	assert.Equal(t, aclTokenRecord{}, tokenRecordFromToken(nil))
	assert.False(t, tokenRecordFromToken(nil).hasGrants())
}

func TestHasGrants(t *testing.T) {
	assert.False(t, aclTokenRecord{}.hasGrants(), "a token granting nothing authorizes nothing")

	tests := map[string]aclTokenRecord{
		"policy":           {Policies: []*consulapi.ACLLink{{Name: "p"}}},
		"role":             {Roles: []*consulapi.ACLLink{{Name: "r"}}},
		"service identity": {ServiceIdentities: []*consulapi.ACLServiceIdentity{{ServiceName: "web"}}},
		"node identity":    {NodeIdentities: []*consulapi.ACLNodeIdentity{{NodeName: "n"}}},
		"templated policy": {TemplatedPolicies: []*consulapi.ACLTemplatedPolicy{{TemplateName: "builtin/service"}}},
	}
	for name, record := range tests {
		t.Run(name, func(t *testing.T) {
			assert.True(t, record.hasGrants())
		})
	}
}

func TestPolicyListFixture(t *testing.T) {
	var entries []*consulapi.ACLPolicyListEntry
	loadFixture(t, "acl-policies.json", &entries)
	require.Len(t, entries, 2)

	byName := map[string]*consulapi.ACLPolicyListEntry{}
	for _, entry := range entries {
		byName[entry.Name] = entry
	}

	unrestricted := byName[globalManagementPolicyName]
	require.NotNil(t, unrestricted)
	assert.Equal(t, globalManagementPolicyID, unrestricted.ID)
	assert.True(t, isGlobalManagementPolicy(unrestricted.ID, unrestricted.Name))

	readOnly := byName["builtin/global-read-only"]
	require.NotNil(t, readOnly)
	assert.False(t, isGlobalManagementPolicy(readOnly.ID, readOnly.Name))

	// the list endpoint carries no rules, which is why the schema reads them
	// through a separate call rather than reporting an empty document. This is
	// asserted against the captured payload, not against the SDK type, because
	// the type having no such field proves nothing about what the agent sent.
	raw, err := os.ReadFile(filepath.Join("testdata", "acl-policies.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "\"Rules\"")
}

func TestRoleListFixture(t *testing.T) {
	var roles []*consulapi.ACLRole
	loadFixture(t, "acl-roles.json", &roles)
	require.Len(t, roles, 1)

	role := roles[0]
	assert.Equal(t, "web-role", role.Name)
	require.Len(t, role.Policies, 1)
	assert.Equal(t, "web-read", role.Policies[0].Name)

	identities := serviceIdentityDicts(role.ServiceIdentities)
	require.Len(t, identities, 1)
	assert.Equal(t, "web", identities[0].(map[string]any)["serviceName"])
}

func TestIdentityDicts(t *testing.T) {
	t.Run("service identities", func(t *testing.T) {
		got := serviceIdentityDicts([]*consulapi.ACLServiceIdentity{
			{ServiceName: "web", Datacenters: []string{"dc1", "dc2"}},
			nil,
			{ServiceName: "db"},
		})
		require.Len(t, got, 2, "nil entries are skipped, not rendered as blanks")
		assert.Equal(t, "web", got[0].(map[string]any)["serviceName"])
		assert.Equal(t, []any{"dc1", "dc2"}, got[0].(map[string]any)["datacenters"])
		// an identity valid everywhere renders an empty list, not null
		assert.Equal(t, []any{}, got[1].(map[string]any)["datacenters"])
	})

	t.Run("node identities", func(t *testing.T) {
		got := nodeIdentityDicts([]*consulapi.ACLNodeIdentity{
			{NodeName: "node-1", Datacenter: "dc1"},
			nil,
		})
		require.Len(t, got, 1)
		assert.Equal(t, "node-1", got[0].(map[string]any)["nodeName"])
		assert.Equal(t, "dc1", got[0].(map[string]any)["datacenter"])
	})

	t.Run("templated policies", func(t *testing.T) {
		got := templatedPolicyDicts([]*consulapi.ACLTemplatedPolicy{
			{
				TemplateName:      "builtin/service",
				TemplateVariables: &consulapi.ACLTemplatedPolicyVariables{Name: "web"},
			},
			// the variables block is optional and must not panic when absent
			{TemplateName: "builtin/dns"},
			nil,
		})
		require.Len(t, got, 2)
		assert.Equal(t, map[string]any{"name": "web"}, got[0].(map[string]any)["templateVariables"])
		assert.Equal(t, map[string]any{}, got[1].(map[string]any)["templateVariables"])
	})

	t.Run("empty input renders an empty list", func(t *testing.T) {
		assert.Equal(t, []any{}, serviceIdentityDicts(nil))
		assert.Equal(t, []any{}, nodeIdentityDicts(nil))
		assert.Equal(t, []any{}, templatedPolicyDicts(nil))
	})
}

func TestNullableTime(t *testing.T) {
	// the zero time must stay null rather than rendering as a date in year one
	assert.Nil(t, nullableTime(time.Time{}))

	moment := time.Date(2026, 8, 19, 13, 4, 0, 0, time.UTC)
	got := nullableTime(moment)
	require.NotNil(t, got)
	assert.Equal(t, moment, *got)
}

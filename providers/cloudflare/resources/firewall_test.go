// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirewallRules(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/firewall/rules", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("firewall_rules"))
	})

	result, err := zone.firewallRules()
	require.NoError(t, err)
	require.Len(t, result, 1)

	rule := result[0].(*mqlCloudflareZoneFirewallRule)
	assert.Equal(t, "fw-rule-1", rule.Id.Data)
	assert.Equal(t, "Block bad bots", rule.Description.Data)
	assert.Equal(t, "block", rule.Action.Data)
	assert.False(t, rule.Paused.Data)
	assert.Equal(t, "(cf.client.bot)", rule.FilterExpression.Data)
	assert.Len(t, rule.Products.Data, 2)
	assert.False(t, rule.CreatedAt.Data.IsZero())
	assert.False(t, rule.UpdatedAt.Data.IsZero())
}

func TestRulesets(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("rulesets"))
	})

	result, err := zone.rulesets()
	require.NoError(t, err)
	require.Len(t, result, 2)

	rs := result[0].(*mqlCloudflareZoneRuleset)
	assert.Equal(t, "rs-managed-1", rs.Id.Data)
	assert.Equal(t, "Cloudflare Managed Ruleset", rs.Name.Data)
	assert.Equal(t, "managed", rs.Kind.Data)
	assert.Equal(t, "http_request_firewall_managed", rs.Phase.Data)
	assert.Equal(t, "3", rs.Version.Data)

	rs2 := result[1].(*mqlCloudflareZoneRuleset)
	assert.Equal(t, "Custom WAF Rules", rs2.Name.Data)
	assert.Equal(t, "zone", rs2.Kind.Data)
}

func TestPageRules(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/pagerules", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("page_rules"))
	})

	result, err := zone.pageRules()
	require.NoError(t, err)
	require.Len(t, result, 1)

	pr := result[0].(*mqlCloudflareZonePageRule)
	assert.Equal(t, "pr-001", pr.Id.Data)
	assert.Equal(t, "active", pr.Status.Data)
	assert.Equal(t, int64(1), pr.Priority.Data)
	assert.False(t, pr.CreatedAt.Data.IsZero())
}

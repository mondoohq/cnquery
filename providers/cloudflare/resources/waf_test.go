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

func TestWafRules(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("rulesets"))
	})

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets/rs-managed-1", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("waf_ruleset_managed"))
	})

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets/rs-custom-1", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("waf_ruleset_custom"))
	})

	result, err := zone.wafRules()
	require.NoError(t, err)
	require.Len(t, result, 3, "expected 2 rules from managed ruleset + 1 from custom")

	// First rule: high-threat block from managed ruleset
	r1 := result[0].(*mqlCloudflareZoneWafRule)
	assert.Equal(t, "rule-mgd-001", r1.Id.Data)
	assert.Equal(t, "rs-managed-1", r1.RulesetId.Data)
	assert.Equal(t, "Cloudflare Managed Ruleset", r1.RulesetName.Data)
	assert.Equal(t, "managed", r1.RulesetKind.Data)
	assert.Equal(t, "http_request_firewall_managed", r1.RulesetPhase.Data)
	assert.Equal(t, "block", r1.Action.Data)
	assert.Equal(t, "cf.threat_score gt 50", r1.Expression.Data)
	assert.True(t, r1.Enabled.Data)
	assert.Equal(t, int64(50), r1.ScoreThreshold.Data)
	assert.Equal(t, "3", r1.Version.Data)

	// Second rule: disabled challenge from managed ruleset
	r2 := result[1].(*mqlCloudflareZoneWafRule)
	assert.Equal(t, "rule-mgd-002", r2.Id.Data)
	assert.Equal(t, "challenge", r2.Action.Data)
	assert.False(t, r2.Enabled.Data, "rule with enabled=false should preserve that")

	// Third rule: zone-defined custom rule
	r3 := result[2].(*mqlCloudflareZoneWafRule)
	assert.Equal(t, "rule-custom-001", r3.Id.Data)
	assert.Equal(t, "rs-custom-1", r3.RulesetId.Data)
	assert.Equal(t, "zone", r3.RulesetKind.Data)
	assert.Equal(t, "block", r3.Action.Data)
	assert.True(t, r3.Enabled.Data)
}

func TestWafRules_skipsForbiddenRulesets(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("rulesets"))
	})

	// Managed ruleset returns 403 — should be skipped, not failed
	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets/rs-managed-1", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		jsonResponse(w, `{"success":false,"errors":[{"code":10000,"message":"Insufficient permissions"}]}`)
	})

	// Custom ruleset succeeds
	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rulesets/rs-custom-1", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, loadFixture("waf_ruleset_custom"))
	})

	result, err := zone.wafRules()
	require.NoError(t, err)
	require.Len(t, result, 1, "should keep going past forbidden managed ruleset")
	r := result[0].(*mqlCloudflareZoneWafRule)
	assert.Equal(t, "rule-custom-001", r.Id.Data)
}

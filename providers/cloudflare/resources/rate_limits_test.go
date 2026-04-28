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

func TestRateLimitRules(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/zones/%s/rate_limits", testZoneID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("rate_limit_rules"))
	})

	result, err := zone.rateLimitRules()
	require.NoError(t, err)
	require.Len(t, result, 2)

	// First rule — challenge-mode, no response body, status filter
	r1 := result[0].(*mqlCloudflareZoneRateLimitRule)
	assert.Equal(t, "rl-001", r1.Id.Data)
	assert.Equal(t, "Login burst protection", r1.Description.Data)
	assert.False(t, r1.Disabled.Data)
	assert.Equal(t, int64(5), r1.Threshold.Data)
	assert.Equal(t, int64(60), r1.Period.Data)
	assert.Equal(t, "*example.com/login", r1.UrlPattern.Data)
	assert.Equal(t, []any{"POST"}, r1.Methods.Data)
	assert.Equal(t, []any{"HTTPS"}, r1.Schemes.Data)
	assert.Equal(t, []any{int64(401), int64(403)}, r1.ResponseStatuses.Data)

	a1, ok := r1.Action.Data.(map[string]any)
	require.True(t, ok, "action should decode to map[string]any")
	assert.Equal(t, "challenge", a1["mode"])
	assert.Equal(t, int64(300), a1["timeout"])
	assert.Nil(t, a1["response"], "no custom response → response key absent")

	// Second rule — disabled, simulate-mode, custom response body
	r2 := result[1].(*mqlCloudflareZoneRateLimitRule)
	assert.Equal(t, "rl-002", r2.Id.Data)
	assert.True(t, r2.Disabled.Data)
	assert.Equal(t, int64(100), r2.Threshold.Data)

	a2, ok := r2.Action.Data.(map[string]any)
	require.True(t, ok, "action should decode to map[string]any")
	assert.Equal(t, "simulate", a2["mode"])
	resp, ok := a2["response"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "application/json", resp["contentType"])
	assert.Contains(t, resp["body"].(string), "rate_limited")
}

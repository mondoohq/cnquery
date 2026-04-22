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

func TestGatewayRules(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/gateway/rules", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("gateway_rules"))
	})

	result, err := one.gatewayRules()
	require.NoError(t, err)
	require.Len(t, result, 1)

	rule := result[0].(*mqlCloudflareOneGatewayRule)
	assert.Equal(t, "gw-rule-001", rule.Id.Data)
	assert.Equal(t, "Block Malware Domains", rule.Name.Data)
	assert.Equal(t, "block", rule.Action.Data)
	assert.True(t, rule.Enabled.Data)
	assert.Equal(t, int64(1), rule.Precedence.Data)
	assert.Equal(t, "dns.fqdn in $malware_domains", rule.Traffic.Data)
	assert.Equal(t, int64(2), rule.Version.Data)
	assert.Len(t, rule.Filters.Data, 1)
}

func TestLists(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/gateway/lists", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("teams_lists"))
	})

	result, err := one.lists()
	require.NoError(t, err)
	require.Len(t, result, 1)

	list := result[0].(*mqlCloudflareOneList)
	assert.Equal(t, "list-001", list.Id.Data)
	assert.Equal(t, "Blocked IPs", list.Name.Data)
	assert.Equal(t, "IP", list.Type.Data)
	assert.Equal(t, int64(150), list.Count.Data)
}

func TestLocations(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/gateway/locations", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("teams_locations"))
	})

	result, err := one.locations()
	require.NoError(t, err)
	require.Len(t, result, 1)

	loc := result[0].(*mqlCloudflareOneLocation)
	assert.Equal(t, "loc-001", loc.Id.Data)
	assert.Equal(t, "Office HQ", loc.Name.Data)
	assert.Equal(t, "office-hq", loc.DohSubdomain.Data)
	assert.Equal(t, "203.0.113.1", loc.Ip.Data)
	assert.True(t, loc.ClientDefault.Data)
	assert.True(t, loc.EcsSupport.Data)
}

func TestDlpProfiles(t *testing.T) {
	env := setupTestEnv(t)
	one := createTestOne(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/dlp/profiles", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("dlp_profiles"))
	})

	result, err := one.dlpProfiles()
	require.NoError(t, err)
	require.Len(t, result, 1)

	profile := result[0].(*mqlCloudflareOneDlpProfile)
	assert.Equal(t, "dlp-001", profile.Id.Data)
	assert.Equal(t, "Credit Card Detection", profile.Name.Data)
	assert.Equal(t, "predefined", profile.Type.Data)
	assert.True(t, profile.OcrEnabled.Data)
	assert.Equal(t, int64(0), profile.AllowedMatchCount.Data)
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNetworkZoneParsing_AllTypes verifies that our custom NetworkZone struct
// handles all three Okta zone types: IP, DYNAMIC, and DYNAMIC_V2.
// The upstream SDK (v2) fails on DYNAMIC_V2 because it expects locations/asns
// to be arrays, but the API returns objects with include/exclude.
func TestNetworkZoneParsing_AllTypes(t *testing.T) {
	data := `[
		{
			"type": "IP",
			"id": "nzowc1U5Jh5xuAK0o0g3",
			"name": "MyIpZone",
			"status": "ACTIVE",
			"usage": "POLICY",
			"system": false,
			"gateways": [{"type": "CIDR", "value": "1.2.3.4/24"}],
			"proxies": [{"type": "RANGE", "value": "3.3.4.5-3.3.4.15"}]
		},
		{
			"type": "DYNAMIC",
			"id": "nzoy0ox5xADOZtKrh0g6",
			"name": "DynamicZone",
			"status": "ACTIVE",
			"usage": "POLICY",
			"system": false,
			"locations": [{"country": "AF", "region": "AF-BGL"}],
			"proxyType": "ANY",
			"asns": ["23457"]
		},
		{
			"type": "DYNAMIC_V2",
			"id": "nzok0oz2xYHOZtIch0g4",
			"name": "DefaultEnhancedDynamicZone",
			"status": "ACTIVE",
			"usage": "BLOCKLIST",
			"system": true,
			"locations": {"include": [{"country": "US", "region": "US-CA"}], "exclude": []},
			"asns": {"include": ["12345"], "exclude": ["99999"]},
			"ipServiceCategories": {"include": ["ALL_ANONYMIZERS"], "exclude": []}
		}
	]`

	var zones []*NetworkZone
	err := json.Unmarshal([]byte(data), &zones)
	require.NoError(t, err)
	require.Len(t, zones, 3)

	// IP zone
	assert.Equal(t, "IP", zones[0].Type)
	assert.Equal(t, "MyIpZone", zones[0].Name)

	gateways, err := NormalizeArrayField(zones[0].Gateways)
	require.NoError(t, err)
	require.Len(t, gateways, 1)
	gw := gateways[0].(map[string]any)
	assert.Equal(t, "CIDR", gw["type"])
	assert.Equal(t, "1.2.3.4/24", gw["value"])

	proxies, err := NormalizeArrayField(zones[0].Proxies)
	require.NoError(t, err)
	require.Len(t, proxies, 1)

	// IP zones have no locations/asns
	locs, err := NormalizeArrayField(zones[0].Locations)
	require.NoError(t, err)
	assert.Nil(t, locs)

	asns, err := NormalizeStringArrayField(zones[0].Asns)
	require.NoError(t, err)
	assert.Nil(t, asns)

	// DYNAMIC zone — locations is a flat array, asns is a flat string array
	assert.Equal(t, "DYNAMIC", zones[1].Type)

	locs, err = NormalizeArrayField(zones[1].Locations)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	loc := locs[0].(map[string]any)
	assert.Equal(t, "AF", loc["country"])
	assert.Equal(t, "AF-BGL", loc["region"])

	asns, err = NormalizeStringArrayField(zones[1].Asns)
	require.NoError(t, err)
	assert.Equal(t, []string{"23457"}, asns)

	// DYNAMIC_V2 zone — the problematic case. Locations and asns are objects.
	assert.Equal(t, "DYNAMIC_V2", zones[2].Type)
	assert.Equal(t, "DefaultEnhancedDynamicZone", zones[2].Name)
	assert.True(t, *zones[2].System)

	locs, err = NormalizeArrayField(zones[2].Locations)
	require.NoError(t, err)
	require.Len(t, locs, 1, "DYNAMIC_V2 locations.include should be extracted")
	loc = locs[0].(map[string]any)
	assert.Equal(t, "US", loc["country"])
	assert.Equal(t, "US-CA", loc["region"])

	asns, err = NormalizeStringArrayField(zones[2].Asns)
	require.NoError(t, err)
	assert.Equal(t, []string{"12345"}, asns, "DYNAMIC_V2 asns.include should be extracted")
}

func TestNormalizeArrayField_Null(t *testing.T) {
	result, err := NormalizeArrayField(json.RawMessage("null"))
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestNormalizeArrayField_Empty(t *testing.T) {
	result, err := NormalizeArrayField(json.RawMessage("[]"))
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestNormalizeArrayField_EmptyIncludeExclude(t *testing.T) {
	result, err := NormalizeArrayField(json.RawMessage(`{"include": [], "exclude": []}`))
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestNormalizeStringArrayField_IncludeExclude(t *testing.T) {
	result, err := NormalizeStringArrayField(json.RawMessage(`{"include": ["a", "b"], "exclude": ["c"]}`))
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestNormalizeArrayField_InvalidJSON(t *testing.T) {
	_, err := NormalizeArrayField(json.RawMessage(`{invalid`))
	assert.Error(t, err)
}

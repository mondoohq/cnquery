// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlowLogAnalyticsConfigJSONTags guards the JSON tags on the flow-log
// analytics struct. They were previously scrambled — `enabled` serialized
// under `allowedApplications`, and `workspaceId`/`workspaceResourceId` swapped
// — so any audit reading those dict fields got the wrong value.
func TestFlowLogAnalyticsConfigJSONTags(t *testing.T) {
	cfg := flowLogAnalyticsConfig{
		Enabled:             true,
		AnalyticsInterval:   10,
		WorkspaceId:         "ws-id",
		WorkspaceResourceId: "ws-resource-id",
		WorkspaceRegion:     "westus",
	}
	b, err := json.Marshal(cfg)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, true, m["enabled"])
	assert.Equal(t, float64(10), m["analyticsInterval"])
	assert.Equal(t, "ws-id", m["workspaceId"])
	assert.Equal(t, "ws-resource-id", m["workspaceResourceId"])
	assert.Equal(t, "westus", m["workspaceRegion"])

	// none of the old/scrambled keys should be present
	for _, badKey := range []string{"allowedApplications"} {
		_, present := m[badKey]
		assert.False(t, present, "unexpected scrambled key %q", badKey)
	}
}

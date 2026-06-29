// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// rootField fetches a single field off the singleton vsphere resource and
// returns the data response, failing the test on any resolver error.
func rootField(t *testing.T, srv *Service, connID uint32, field string) *plugin.DataRes {
	t.Helper()

	root, err := srv.GetData(&plugin.DataReq{
		Connection: connID,
		Resource:   "vsphere",
	})
	require.NoError(t, err)
	rootID := string(root.Data.Value)

	resp, err := srv.GetData(&plugin.DataReq{
		Connection: connID,
		Resource:   "vsphere",
		ResourceId: rootID,
		Field:      field,
	})
	require.NoError(t, err, "resolving vsphere.%s should not error", field)
	return resp
}

// TestGovernanceResources exercises every governance/inventory-metadata root
// accessor against the vcsim simulator. The simulator does not implement the
// vAPI certificate-management endpoints, so vsphere.certificates is expected to
// resolve cleanly to an empty list rather than error (the resolver treats those
// endpoints as best-effort).
func TestGovernanceResources(t *testing.T) {
	vs, srv, connRes := newTestService()
	defer vs.Close()

	for _, field := range []string{
		"categories",
		"tags",
		"contentLibraries",
		"customFields",
		"alarms",
		"triggeredAlarms",
		"scheduledTasks",
		"recentTasks",
		"events",
		"certificates",
	} {
		t.Run(field, func(t *testing.T) {
			resp := rootField(t, srv, connRes.Id, field)
			require.NotNil(t, resp.Data)
		})
	}

	// vcsim ships the default vCenter alarm definitions, so alarms should be
	// non-empty and prove the resolver maps real data through the runtime.
	// Distinct resource references additionally prove each alarm gets a unique
	// __id (a missing __id would collapse every entry onto one cache key).
	t.Run("alarms are populated and uniquely identified", func(t *testing.T) {
		resp := rootField(t, srv, connRes.Id, "alarms")
		require.Greater(t, len(resp.Data.Array), 1, "expected multiple default vCenter alarms")

		ids := map[string]struct{}{}
		for _, a := range resp.Data.Array {
			ids[string(a.Value)] = struct{}{}
		}
		require.Equal(t, len(resp.Data.Array), len(ids), "every alarm must have a unique __id")
	})

	// vcsim records events for the operations it performs at startup.
	t.Run("events are populated", func(t *testing.T) {
		resp := rootField(t, srv, connRes.Id, "events")
		require.NotEmpty(t, resp.Data.Array, "expected simulator events")
	})
}

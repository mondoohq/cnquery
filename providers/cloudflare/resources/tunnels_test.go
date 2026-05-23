// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTunnels(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/cfd_tunnel", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("tunnels"))
	})

	result, err := zone.tunnels()
	require.NoError(t, err)
	require.Len(t, result, 2)

	tunnel := result[0].(*mqlCloudflareTunnel)
	assert.Equal(t, "f70ff985-a4ef-4643-bbbc-4a0ed4fc8415", tunnel.Id.Data)
	assert.Equal(t, "blog-tunnel", tunnel.Name.Data)
	assert.Equal(t, "cfd_tunnel", tunnel.TunnelType.Data)
	assert.Equal(t, "healthy", tunnel.Status.Data)
	assert.True(t, tunnel.RemoteConfig.Data)
	assert.False(t, tunnel.CreatedAt.Data.IsZero())

	// Verify connections
	require.Len(t, tunnel.Connections.Data, 1)
	conn := tunnel.Connections.Data[0].(*mqlCloudflareTunnelConnection)
	assert.Equal(t, "conn-1234", conn.Id.Data)
	assert.Equal(t, "DFW", conn.ColoName.Data)
	assert.Equal(t, "198.51.100.1", conn.OriginIp.Data)
	assert.False(t, conn.IsPendingReconnect.Data)
	assert.False(t, conn.OpenedAt.Data.IsZero())

	// Second tunnel has no connections
	tunnel2 := result[1].(*mqlCloudflareTunnel)
	assert.Equal(t, "api-tunnel", tunnel2.Name.Data)
	assert.Equal(t, "down", tunnel2.Status.Data)
	assert.Len(t, tunnel2.Connections.Data, 0)
}

func TestTunnelRoutes(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/teamnet/routes", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("tunnel_routes"))
	})

	result, err := zone.tunnelRoutes()
	require.NoError(t, err)
	require.Len(t, result, 1)

	route := result[0].(*mqlCloudflareTunnelRoute)
	assert.Equal(t, "10.0.0.0/8", route.Network.Data)
	assert.Equal(t, "f70ff985-a4ef-4643-bbbc-4a0ed4fc8415", route.TunnelId.Data)
	assert.Equal(t, "blog-tunnel", route.TunnelName.Data)
	assert.Equal(t, "Internal network", route.Comment.Data)
}

func TestTunnelVirtualNetworks(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/teamnet/virtual_networks", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, loadFixture("tunnel_virtual_networks"))
	})

	result, err := zone.tunnelVirtualNetworks()
	require.NoError(t, err)
	require.Len(t, result, 1)

	vnet := result[0].(*mqlCloudflareTunnelVirtualNetwork)
	assert.Equal(t, "vnet-1234", vnet.Id.Data)
	assert.Equal(t, "default-vnet", vnet.Name.Data)
	assert.True(t, vnet.IsDefaultNetwork.Data)
	assert.Equal(t, "Default virtual network", vnet.Comment.Data)
}

// TestTunnelRoutesVnetIDInCacheKey verifies that two routes advertising the
// same network through the same tunnel from two different virtual networks
// produce two distinct rows (rather than collapsing into one cached entry).
// The previous __id `tunnelRoute@<network>@<tunnelId>` collided in this case.
func TestTunnelRoutesVnetIDInCacheKey(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/teamnet/routes", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, `{
			"success": true, "errors": [], "messages": [],
			"result": [
				{"network": "10.0.0.0/8", "tunnel_id": "tun-1", "tunnel_name": "t", "comment": "via vnet-a", "virtual_network_id": "vnet-a", "created_at": "2024-01-01T00:00:00Z"},
				{"network": "10.0.0.0/8", "tunnel_id": "tun-1", "tunnel_name": "t", "comment": "via vnet-b", "virtual_network_id": "vnet-b", "created_at": "2024-01-01T00:00:00Z"}
			]
		}`)
	})

	result, err := zone.tunnelRoutes()
	require.NoError(t, err)
	require.Len(t, result, 2, "routes with distinct virtual_network_id must not collapse into one")

	r1 := result[0].(*mqlCloudflareTunnelRoute)
	r2 := result[1].(*mqlCloudflareTunnelRoute)

	// Same (network, tunnelId) but distinct vnet IDs.
	assert.Equal(t, r1.Network.Data, r2.Network.Data)
	assert.Equal(t, r1.TunnelId.Data, r2.TunnelId.Data)
	assert.NotEqual(t, r1.cacheVirtualNetworkID, r2.cacheVirtualNetworkID)
	assert.NotEqual(t, r1.Comment.Data, r2.Comment.Data, "each row must keep its own data, not share with its collision peer")
}

// TestTunnelRoutesPagination guards against the previously single-page call
// silently truncating large accounts. The handler returns full-size pages
// (perPage=50) twice, then a short page, and we assert the loop consumed all
// three pages and bumped page numbers monotonically.
func TestTunnelRoutesPagination(t *testing.T) {
	env := setupTestEnv(t)
	zone := createTestZone(t, env)

	const perPage = 50
	var calls int32

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/teamnet/routes", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		count := perPage
		if page >= 3 {
			count = 7 // short page → terminates loop
		}
		fmt.Fprint(w, `{"success": true, "errors": [], "messages": [], "result": [`)
		for i := 0; i < count; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"network": "10.%d.%d.0/24", "tunnel_id": "t-p%d-i%d", "tunnel_name": "t", "comment": "", "virtual_network_id": "v-p%d-i%d", "created_at": "2024-01-01T00:00:00Z"}`, page, i, page, i, page, i)
		}
		fmt.Fprint(w, `]}`)
	})

	result, err := zone.tunnelRoutes()
	require.NoError(t, err)
	require.Equal(t, perPage*2+7, len(result), "all three pages must be consumed")
	require.Equal(t, int32(3), atomic.LoadInt32(&calls), "exactly three calls (page=1,2,3)")
}

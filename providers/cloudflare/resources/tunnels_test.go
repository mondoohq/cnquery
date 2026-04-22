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

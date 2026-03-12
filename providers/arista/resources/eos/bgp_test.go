// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBGPSummaryParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/bgp-summary.json")
	require.NoError(t, err)

	var summary showIPBgpSummary
	err = json.Unmarshal(data, &summary)
	require.NoError(t, err)

	// Verify VRF count
	assert.Len(t, summary.VRFs, 2)

	// Verify default VRF
	defaultVrf, ok := summary.VRFs["default"]
	require.True(t, ok)
	assert.Equal(t, "10.0.0.1", defaultVrf.RouterID)
	assert.Equal(t, int64(65001), defaultVrf.ASN)
	assert.Len(t, defaultVrf.Peers, 2)

	// Verify established peer
	peer1, ok := defaultVrf.Peers["10.0.0.2"]
	require.True(t, ok)
	assert.Equal(t, "Established", peer1.PeerState)
	assert.Equal(t, "65002", peer1.ASN)
	assert.Equal(t, int64(150), peer1.PrefixAccepted)
	assert.Equal(t, int64(200), peer1.PrefixReceived)
	assert.InDelta(t, 86400.5, peer1.UpDownTime, 0.1)
	assert.False(t, peer1.UnderMaintenance)

	// Verify idle peer
	peer2, ok := defaultVrf.Peers["10.0.0.3"]
	require.True(t, ok)
	assert.Equal(t, "Idle", peer2.PeerState)
	assert.Equal(t, int64(0), peer2.PrefixAccepted)
	assert.Equal(t, int64(0), peer2.PrefixReceived)

	// Verify MGMT VRF
	mgmtVrf, ok := summary.VRFs["MGMT"]
	require.True(t, ok)
	assert.Equal(t, "172.16.0.1", mgmtVrf.RouterID)
	assert.Len(t, mgmtVrf.Peers, 1)

	mgmtPeer, ok := mgmtVrf.Peers["172.16.0.2"]
	require.True(t, ok)
	assert.True(t, mgmtPeer.UnderMaintenance)
	assert.Equal(t, int64(5), mgmtPeer.PrefixAccepted)
}

func TestBGPNeighborsParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/bgp-neighbors.json")
	require.NoError(t, err)

	var neighbors showIPBgpNeighbors
	err = json.Unmarshal(data, &neighbors)
	require.NoError(t, err)

	// Verify VRF count
	assert.Len(t, neighbors.VRFs, 2)

	// Verify default VRF peers
	defaultVrf, ok := neighbors.VRFs["default"]
	require.True(t, ok)
	assert.Len(t, defaultVrf.PeerList, 2)

	// Verify first peer details
	peer1 := defaultVrf.PeerList[0]
	assert.Equal(t, "10.0.0.2", peer1.PeerAddress)
	assert.Equal(t, "upstream-router", peer1.Description)
	assert.Equal(t, "65002", peer1.ASN)
	assert.Equal(t, "IMPORT-FILTER", peer1.InboundRouteMap)
	assert.Equal(t, "EXPORT-FILTER", peer1.OutboundRouteMap)

	// Verify peer with no route maps
	peer2 := defaultVrf.PeerList[1]
	assert.Equal(t, "10.0.0.3", peer2.PeerAddress)
	assert.Equal(t, "backup-router", peer2.Description)
	assert.Empty(t, peer2.InboundRouteMap)
	assert.Empty(t, peer2.OutboundRouteMap)

	// Verify MGMT VRF
	mgmtVrf, ok := neighbors.VRFs["MGMT"]
	require.True(t, ok)
	assert.Len(t, mgmtVrf.PeerList, 1)
	assert.Equal(t, "172.16.0.2", mgmtVrf.PeerList[0].PeerAddress)
	assert.Equal(t, "MGMT-IN", mgmtVrf.PeerList[0].InboundRouteMap)
}

func TestBGPSummaryGetCmd(t *testing.T) {
	s := &showIPBgpSummary{}
	assert.Equal(t, "show ip bgp summary", s.GetCmd())
}

func TestBGPNeighborsGetCmd(t *testing.T) {
	s := &showIPBgpNeighbors{}
	assert.Equal(t, "show ip bgp neighbors", s.GetCmd())
}

func TestBGPSummaryEmptyVRFs(t *testing.T) {
	jsonData := `{"vrfs": {}}`
	var summary showIPBgpSummary
	err := json.Unmarshal([]byte(jsonData), &summary)
	require.NoError(t, err)
	assert.Empty(t, summary.VRFs)
}

func TestBGPSummaryEmptyPeers(t *testing.T) {
	jsonData := `{"vrfs": {"default": {"routerId": "1.2.3.4", "asn": 100, "peers": {}}}}`
	var summary showIPBgpSummary
	err := json.Unmarshal([]byte(jsonData), &summary)
	require.NoError(t, err)
	assert.Empty(t, summary.VRFs["default"].Peers)
	assert.Equal(t, int64(100), summary.VRFs["default"].ASN)
}

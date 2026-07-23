// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	pp "go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func TestConnectionGraph(t *testing.T) {
	g := newConnectionGraph()
	g.addNode(1, connectReq{})
	g.addNode(2, connectReq{})
	g.addNode(3, connectReq{})
	g.addNode(4, connectReq{})

	g.setEdge(4, 2)
	g.setEdge(2, 1)
	g.setEdge(3, 1)

	sorted := g.topoSort()

	require.Len(t, sorted, 4)

	requireComesBefore := func(t *testing.T, sorted []uint32, before, after uint32) {
		beforeIdx := -1
		afterIdx := -1
		for i, n := range sorted {
			if n == before {
				beforeIdx = i
			}
			if n == after {
				afterIdx = i
			}
		}
		require.True(t, beforeIdx >= 0, "before node not found")
		require.True(t, afterIdx >= 0, "after node not found")
		require.True(t, beforeIdx < afterIdx, "before node does not come before after node")
	}

	requireComesBefore(t, sorted, 2, 4)
	requireComesBefore(t, sorted, 1, 2)
	requireComesBefore(t, sorted, 1, 3)

	g.markDisconnected(1)
	g.garbageCollect()

	sorted = g.topoSort()
	require.Len(t, sorted, 4)
	requireComesBefore(t, sorted, 2, 4)
	requireComesBefore(t, sorted, 1, 2)
	requireComesBefore(t, sorted, 1, 3)

	g.markDisconnected(2)
	g.garbageCollect()

	sorted = g.topoSort()
	require.Len(t, sorted, 4)
	requireComesBefore(t, sorted, 2, 4)
	requireComesBefore(t, sorted, 1, 2)
	requireComesBefore(t, sorted, 1, 3)

	g.markDisconnected(4)
	g.garbageCollect()

	sorted = g.topoSort()
	require.Len(t, sorted, 2)
	requireComesBefore(t, sorted, 1, 3)
	require.NotContains(t, g.nodes, uint32(2))
	require.NotContains(t, g.nodes, uint32(4))

	g.markDisconnected(3)
	g.garbageCollect()

	sorted = g.topoSort()
	require.Len(t, sorted, 0)
	require.Empty(t, g.nodes)
	require.Empty(t, g.edges)
}

func TestConnectionGraph_CollectedDataPreserved(t *testing.T) {
	g := newConnectionGraph()

	parentData := connectReq{req: &pp.ConnectReq{}}
	g.addNode(1, parentData)
	g.addNode(2, connectReq{})
	g.setEdge(2, 1)

	// Disconnect both — parent first, then child. GC after child
	// disconnect should collect both.
	g.markDisconnected(1)
	g.markDisconnected(2)
	collected := g.garbageCollect()
	require.Len(t, collected, 2)
	require.NotContains(t, g.nodes, uint32(1))
	require.NotContains(t, g.nodes, uint32(2))

	// Collected data should be preserved for reconnection.
	cr, ok := g.getCollectedNode(1)
	require.True(t, ok, "collected parent data should be preserved")
	assert.Equal(t, parentData.req, cr.req)

	cr2, ok := g.getCollectedNode(2)
	require.True(t, ok, "collected child data should be preserved")
	_ = cr2

	// Re-adding a node should clear its collected entry.
	g.addNode(1, parentData)
	_, ok = g.getCollectedNode(1)
	assert.False(t, ok, "collected data should be cleared after re-add")
}

func TestConnectionGraph_ReconnectedParentKeptByChild(t *testing.T) {
	g := newConnectionGraph()

	// Simulate: parent=1 connected, child=2 connected with edge to 1
	g.addNode(1, connectReq{})
	g.addNode(2, connectReq{})
	g.setEdge(2, 1)

	// Disconnect parent, then child — both get GC'd.
	g.markDisconnected(1)
	g.markDisconnected(2)
	g.garbageCollect()
	require.Empty(t, g.nodes)

	// New child=3 arrives. Reconnect parent as implicit node.
	g.addImplicitNode(1, connectReq{})
	g.addNode(3, connectReq{})
	g.setEdge(3, 1)

	// GC should NOT collect the implicit parent — child 3 keeps it alive.
	collected := g.garbageCollect()
	require.Empty(t, collected)
	require.Contains(t, g.nodes, uint32(1))
	require.Contains(t, g.nodes, uint32(3))

	// Disconnect child 3 — now parent should be collected.
	g.markDisconnected(3)
	collected = g.garbageCollect()
	require.Len(t, collected, 2)
	require.NotContains(t, g.nodes, uint32(1))
	require.NotContains(t, g.nodes, uint32(3))
}

// --- End-to-end simulation of the K8s scan pipeline race condition ---

// mockPlugin tracks which connection IDs are currently alive (have a
// runtime) inside the provider process. It mirrors the real Service's
// runtimes map closely enough to reproduce the bug.
type mockPlugin struct {
	mu       sync.Mutex
	runtimes map[uint32]bool // true = runtime alive
	connects int             // total Connect calls (for assertions)
}

func newMockPlugin() *mockPlugin {
	return &mockPlugin{runtimes: make(map[uint32]bool)}
}

func (m *mockPlugin) Connect(req *pp.ConnectReq, _ pp.ProviderCallback) (*pp.ConnectRes, error) {
	if len(req.Asset.GetConnections()) == 0 {
		return &pp.ConnectRes{}, nil
	}
	conf := req.Asset.Connections[0]

	m.mu.Lock()
	defer m.mu.Unlock()
	m.connects++

	// Already alive? Return early (idempotent, like Service.AddRuntime).
	if m.runtimes[conf.Id] {
		return &pp.ConnectRes{}, nil
	}

	// Check parent, mimicking real AddRuntime before our safety-net fix.
	if conf.ParentConnectionId > 0 {
		if !m.runtimes[conf.ParentConnectionId] {
			return nil, fmt.Errorf("parent connection %d not found", conf.ParentConnectionId)
		}
	}

	m.runtimes[conf.Id] = true
	return &pp.ConnectRes{}, nil
}

func (m *mockPlugin) Disconnect(req *pp.DisconnectReq) (*pp.DisconnectRes, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runtimes, req.Connection)
	return &pp.DisconnectRes{}, nil
}

func (m *mockPlugin) alive(id uint32) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runtimes[id]
}

// Stubs — not exercised by these tests.
func (m *mockPlugin) Heartbeat(*pp.HeartbeatReq) (*pp.HeartbeatRes, error) {
	return &pp.HeartbeatRes{}, nil
}
func (m *mockPlugin) ParseCLI(*pp.ParseCLIReq) (*pp.ParseCLIRes, error) { return nil, nil }
func (m *mockPlugin) MockConnect(*pp.ConnectReq, pp.ProviderCallback) (*pp.ConnectRes, error) {
	return nil, nil
}

func (m *mockPlugin) Shutdown(*pp.ShutdownReq) (*pp.ShutdownRes, error) {
	return &pp.ShutdownRes{}, nil
}
func (m *mockPlugin) GetData(*pp.DataReq) (*pp.DataRes, error)     { return nil, nil }
func (m *mockPlugin) StoreData(*pp.StoreReq) (*pp.StoreRes, error) { return nil, nil }

// connectReqFor builds a ConnectReq for a connection with the given ID
// and optional parent.
func connectReqFor(id, parentId uint32) *pp.ConnectReq {
	return &pp.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{{
				Id:                 id,
				ParentConnectionId: parentId,
			}},
		},
	}
}

// TestRestartableProvider_ParentReconnectsAfterGC reproduces the exact
// K8s scan pipeline race:
//
//  1. Namespace connections (parents) are created during stage-2 discovery.
//  2. Discovery finishes → namespace connections are disconnected → GC'd.
//  3. Workload connections (children) arrive in a later batch with
//     ParentConnectionId pointing at the GC'd namespace.
//
// Before the fix, step 3 fails with "parent connection N not found".
// After the fix, the parent is transparently reconnected from preserved
// data.
func TestRestartableProvider_ParentReconnectsAfterGC(t *testing.T) {
	mock := newMockPlugin()
	rp := &RestartableProvider{
		plugin:          mock,
		connectionGraph: newConnectionGraph(),
	}

	const (
		nsConn         = uint32(2)  // namespace connection (parent)
		workloadBatch1 = uint32(10) // first batch child
		workloadBatch2 = uint32(20) // second batch child (arrives after GC)
	)

	// --- Stage 2 discovery: connect namespace ---
	_, err := rp.Connect(connectReqFor(nsConn, 0), nil)
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn))

	// --- Discovery done: disconnect namespace ---
	// This is the real flow: discovery finishes and the namespace is
	// disconnected BEFORE any workloads connect. Since no children
	// exist in the graph, GC immediately collects the namespace.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: nsConn})
	require.NoError(t, err)
	require.False(t, mock.alive(nsConn), "namespace runtime should be disconnected")

	// --- First batch: connect a workload child ---
	// The parent was GC'd — our fix should reconnect it transparently.
	_, err = rp.Connect(connectReqFor(workloadBatch1, nsConn), nil)
	require.NoError(t, err, "first child should connect after parent is reconnected")
	require.True(t, mock.alive(nsConn), "parent should be alive after reconnection")

	// --- First batch done: disconnect workload ---
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: workloadBatch1})
	require.NoError(t, err)
	// The implicitly reconnected parent should be GC'd again.
	require.False(t, mock.alive(nsConn), "parent should be GC'd after all children disconnect")

	// --- Second batch arrives: another reconnection cycle ---
	_, err = rp.Connect(connectReqFor(workloadBatch2, nsConn), nil)
	require.NoError(t, err, "second child should also connect after parent is re-reconnected")
	require.True(t, mock.alive(nsConn), "parent should be alive again")
	require.True(t, mock.alive(workloadBatch2))
}

// TestRestartableProvider_ParentDisconnectedWhileChildrenActive reproduces
// the bug where a parent is disconnected while children still reference it.
// GC keeps the parent in the graph (children hold it alive), but the
// unconditional r.plugin.Disconnect call removes its Service-layer runtime.
// A subsequent child then finds the parent in the graph (no reconnection),
// but Service.AddRuntime fails to find the parent runtime → warning.
func TestRestartableProvider_ParentDisconnectedWhileChildrenActive(t *testing.T) {
	mock := newMockPlugin()
	rp := &RestartableProvider{
		plugin:          mock,
		connectionGraph: newConnectionGraph(),
	}

	const (
		nsConn = uint32(2)
		child1 = uint32(10)
		child2 = uint32(11)
		child3 = uint32(12)
	)

	// Connect namespace.
	_, err := rp.Connect(connectReqFor(nsConn, 0), nil)
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn))

	// Connect first child (workload).
	_, err = rp.Connect(connectReqFor(child1, nsConn), nil)
	require.NoError(t, err)
	require.True(t, mock.alive(child1))

	// Disconnect namespace while child1 is still connected.
	// GC should keep node nsConn (child1 references it).
	// The Service-layer runtime for nsConn must stay alive.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: nsConn})
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn),
		"parent runtime must stay alive while children reference it")

	// Connect second child — parent is in graph, no reconnection needed.
	// This MUST succeed because the parent runtime is still alive.
	_, err = rp.Connect(connectReqFor(child2, nsConn), nil)
	require.NoError(t, err, "child2 should connect with parent still alive in Service")

	// Disconnect child1 — parent still needed by child2.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: child1})
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn), "parent still needed by child2")

	// Disconnect child2 — parent no longer needed, should be GC'd.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: child2})
	require.NoError(t, err)
	require.False(t, mock.alive(nsConn), "parent should be GC'd when all children disconnect")

	// A third child arrives after full GC — should reconnect via collectedNodes.
	_, err = rp.Connect(connectReqFor(child3, nsConn), nil)
	require.NoError(t, err, "child3 should connect after parent is re-reconnected from collectedNodes")
	require.True(t, mock.alive(nsConn))
}

// TestRestartableProvider_OverlappingBatchesWithParentDisconnect simulates
// the realistic scanner pattern where batches overlap: some children from
// batch 1 are still connected when the parent is disconnected, and new
// children from batch 2 arrive afterward.
func TestRestartableProvider_OverlappingBatchesWithParentDisconnect(t *testing.T) {
	mock := newMockPlugin()
	rp := &RestartableProvider{
		plugin:          mock,
		connectionGraph: newConnectionGraph(),
	}

	const nsConn = uint32(2)

	// Connect namespace.
	_, err := rp.Connect(connectReqFor(nsConn, 0), nil)
	require.NoError(t, err)

	// Batch 1: connect children 10, 11, 12.
	for _, id := range []uint32{10, 11, 12} {
		_, err := rp.Connect(connectReqFor(id, nsConn), nil)
		require.NoError(t, err)
	}

	// Scanner closes namespace (done scanning it as an asset).
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: nsConn})
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn), "parent kept alive by batch 1 children")

	// Batch 1 children finish one by one.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: 10})
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn), "parent kept alive by children 11, 12")

	// Batch 2 arrives while batch 1 isn't fully done yet.
	for _, id := range []uint32{13, 14} {
		_, err := rp.Connect(connectReqFor(id, nsConn), nil)
		require.NoError(t, err, "batch 2 child %d should connect", id)
	}

	// Finish remaining batch 1 children.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: 11})
	require.NoError(t, err)
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: 12})
	require.NoError(t, err)
	require.True(t, mock.alive(nsConn), "parent kept alive by batch 2 children")

	// Finish batch 2.
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: 13})
	require.NoError(t, err)
	_, err = rp.Disconnect(&pp.DisconnectReq{Connection: 14})
	require.NoError(t, err)
	require.False(t, mock.alive(nsConn), "parent GC'd after all children disconnect")

	// Batch 3 arrives after full GC — reconnection from collectedNodes.
	_, err = rp.Connect(connectReqFor(15, nsConn), nil)
	require.NoError(t, err, "batch 3 child should reconnect parent from collectedNodes")
	require.True(t, mock.alive(nsConn))
}

// TestRestartableProvider_MultiNamespaceMultiBatch simulates a realistic
// K8s cluster scan: 5 namespaces × 10 workloads each, processed in
// batches of 15. This catches ordering-dependent bugs that single-pair
// tests miss.
func TestRestartableProvider_MultiNamespaceMultiBatch(t *testing.T) {
	mock := newMockPlugin()
	rp := &RestartableProvider{
		plugin:          mock,
		connectionGraph: newConnectionGraph(),
	}

	const (
		numNamespaces  = 5
		workloadsPerNs = 10
		batchSize      = 15
	)

	// Assign connection IDs the way the coordinator would: namespaces get
	// low IDs, workloads get higher IDs interleaved with namespace discovery.
	nsID := func(ns int) uint32 { return uint32(ns + 1) }
	wlID := func(ns, wl int) uint32 { return uint32(100 + ns*workloadsPerNs + wl) }

	// --- Discovery phase: connect all namespaces (stage 2) ---
	for ns := range numNamespaces {
		_, err := rp.Connect(connectReqFor(nsID(ns), 0), nil)
		require.NoError(t, err)
	}

	// --- Discovery done: disconnect all namespaces ---
	for ns := range numNamespaces {
		_, err := rp.Disconnect(&pp.DisconnectReq{Connection: nsID(ns)})
		require.NoError(t, err)
	}

	// All namespaces should be GC'd now (no children yet).
	for ns := range numNamespaces {
		require.False(t, mock.alive(nsID(ns)), "namespace %d should be GC'd", ns)
	}

	// --- Scan phase: connect workloads in batches ---
	// Build the full workload list.
	type workload struct{ id, parent uint32 }
	var allWorkloads []workload
	for ns := range numNamespaces {
		for wl := range workloadsPerNs {
			allWorkloads = append(allWorkloads, workload{wlID(ns, wl), nsID(ns)})
		}
	}

	// Process in batches: connect batch, disconnect batch, repeat.
	for batchStart := 0; batchStart < len(allWorkloads); batchStart += batchSize {
		batchEnd := min(batchStart+batchSize, len(allWorkloads))
		batch := allWorkloads[batchStart:batchEnd]

		// Connect entire batch.
		for _, wl := range batch {
			_, err := rp.Connect(connectReqFor(wl.id, wl.parent), nil)
			require.NoError(t, err, "workload %d (parent %d) should connect", wl.id, wl.parent)
		}

		// Disconnect entire batch.
		for _, wl := range batch {
			_, err := rp.Disconnect(&pp.DisconnectReq{Connection: wl.id})
			require.NoError(t, err)
		}
	}

	// All workloads processed successfully — no "parent connection N not
	// found" errors. Verify the provider is clean.
	for ns := range numNamespaces {
		assert.False(t, mock.alive(nsID(ns)), "namespace %d should be GC'd after scan", ns)
	}
}

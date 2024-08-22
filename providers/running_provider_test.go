package providers

import (
	"testing"

	"github.com/stretchr/testify/require"
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

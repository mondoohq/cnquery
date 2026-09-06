// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/recording"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
	"go.uber.org/mock/gomock"
)

// The runtime is what llx dereferences a typed asset reference through.
var _ llx.AssetResolver = (*Runtime)(nil)

// multiAssetRecording is a recording that holds several assets and nothing
// else. Resolution only reads the asset list off it.
type multiAssetRecording struct {
	recording.Null
	assets []*recording.Asset
}

func (m *multiAssetRecording) GetAssetRecordings() []*recording.Asset     { return m.assets }
func (m *multiAssetRecording) SetAssetRecording(uint32, *recording.Asset) {}

func recordedAsset(name string, rels ...*inventory.AssetRelationship) *recording.Asset {
	return &recording.Asset{Asset: &inventory.Asset{
		Name:          name,
		PlatformIds:   []string{"//platformid/" + name},
		Relationships: rels,
	}}
}

func anchoredOn(parent *inventory.Asset, resourceType, resourceID string) *inventory.AssetRelationship {
	return &inventory.AssetRelationship{
		Asset:        parent,
		ResourceType: resourceType,
		ResourceId:   resourceID,
	}
}

func runtimeWith(self *inventory.Asset, assets ...*recording.Asset) *Runtime {
	return &Runtime{
		recording: &multiAssetRecording{assets: assets},
		Provider: &ConnectedProvider{
			Connection: &plugin.ConnectRes{Asset: self},
		},
	}
}

func TestTargetAssetForAnchor(t *testing.T) {
	host := &inventory.Asset{Name: "host", PlatformIds: []string{"//platformid/host"}}
	anchor := &llx.AssetValue{ResourceType: "claude.code.mcpServer", ResourceId: "claude.code.mcpServer/github"}

	t.Run("found through the reverse edge", func(t *testing.T) {
		r := runtimeWith(host,
			recordedAsset("other"),
			recordedAsset("mcp-github", anchoredOn(host, anchor.ResourceType, anchor.ResourceId)),
		)

		target, err := r.targetAssetForAnchor(anchor)
		require.NoError(t, err)
		assert.Equal(t, "mcp-github", target.Name)
		assert.Equal(t, []string{"//platformid/mcp-github"}, target.PlatformIds)
		// A mock connection is what makes this the recorded leg; the live leg
		// is the same call with the target's real connection.
		require.Len(t, target.Connections, 1)
		assert.Equal(t, "mock", target.Connections[0].Type)
	})

	t.Run("a different anchor on the same asset is not a match", func(t *testing.T) {
		r := runtimeWith(host,
			recordedAsset("mcp-github", anchoredOn(host, anchor.ResourceType, "claude.code.mcpServer/other")),
		)

		_, err := r.targetAssetForAnchor(anchor)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no recorded asset is anchored on it")
	})

	// An anchor id is only unique within one parent, so a recording holding two
	// hosts can carry the same anchor twice. The edge naming *this* asset wins.
	t.Run("disambiguated by which asset is named as the parent", func(t *testing.T) {
		otherHost := &inventory.Asset{Name: "other-host", PlatformIds: []string{"//platformid/other-host"}}
		r := runtimeWith(host,
			recordedAsset("mcp-elsewhere", anchoredOn(otherHost, anchor.ResourceType, anchor.ResourceId)),
			recordedAsset("mcp-here", anchoredOn(host, anchor.ResourceType, anchor.ResourceId)),
		)

		target, err := r.targetAssetForAnchor(anchor)
		require.NoError(t, err)
		assert.Equal(t, "mcp-here", target.Name)
	})

	t.Run("ambiguous is an error, not a guess", func(t *testing.T) {
		hostA := &inventory.Asset{Name: "a", PlatformIds: []string{"//platformid/a"}}
		hostB := &inventory.Asset{Name: "b", PlatformIds: []string{"//platformid/b"}}
		r := runtimeWith(host,
			recordedAsset("mcp-a", anchoredOn(hostA, anchor.ResourceType, anchor.ResourceId)),
			recordedAsset("mcp-b", anchoredOn(hostB, anchor.ResourceType, anchor.ResourceId)),
		)

		_, err := r.targetAssetForAnchor(anchor)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "several recorded assets are anchored on it")
	})

	t.Run("no recording to resolve from", func(t *testing.T) {
		r := &Runtime{recording: recording.Null{}}
		_, err := r.targetAssetForAnchor(anchor)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "live connect is not implemented yet")
	})
}

// A per-runtime cache cannot see an A -> B -> A cycle, because each hop is a
// genuine miss on a different runtime. The depth guard is what bounds it, and
// the chain is in the error so the cycle is readable.
func TestAssetResolveDepthGuard(t *testing.T) {
	anchor := &llx.AssetValue{ResourceType: "some.server", ResourceId: "some.server/x"}

	r := runtimeWith(&inventory.Asset{Name: "host"})
	r.resolveChain = []string{"a", "b", "c", "d", "e"}

	_, err := r.runtimeForAnchor(anchor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nested too deeply")
	assert.Contains(t, err.Error(), "a -> b -> c -> d -> e -> some.server (some.server/x)")
}

func TestResolveAssetRootRejectsEmptyInput(t *testing.T) {
	r := runtimeWith(&inventory.Asset{Name: "host"})

	_, err := r.ResolveAssetRoot(nil, "mcp")
	assert.ErrorContains(t, err, "empty asset reference")

	_, err = r.ResolveAssetRoot(&llx.AssetValue{ResourceType: "x", ResourceId: "y"}, "")
	assert.ErrorContains(t, err, "no root resource requested")
}

func TestSameAsset(t *testing.T) {
	assert.True(t, sameAsset(
		&inventory.Asset{Mrn: "//a"}, &inventory.Asset{Mrn: "//a"}))
	assert.True(t, sameAsset(
		&inventory.Asset{PlatformIds: []string{"p1", "p2"}}, &inventory.Asset{PlatformIds: []string{"p2"}}))
	assert.False(t, sameAsset(
		&inventory.Asset{Mrn: "//a"}, &inventory.Asset{Mrn: "//b"}))

	// Two assets that both carry real identifiers and do not match are not the
	// same asset, however their display names read.
	assert.False(t, sameAsset(
		&inventory.Asset{Mrn: "//a", Name: "host"}, &inventory.Asset{Mrn: "//b", Name: "host"}))

	// A stub with neither falls back to the name, which is all it has.
	assert.True(t, sameAsset(&inventory.Asset{Name: "host"}, &inventory.Asset{Name: "host"}))
	assert.False(t, sameAsset(&inventory.Asset{Name: "host"}, &inventory.Asset{Name: "other"}))
	assert.False(t, sameAsset(&inventory.Asset{}, &inventory.Asset{}))
	assert.False(t, sameAsset(nil, &inventory.Asset{Name: "host"}))
}

func TestAnchorLabel(t *testing.T) {
	assert.Equal(t, "a.b (a.b/1)", anchorLabel(&llx.AssetValue{ResourceType: "a.b", ResourceId: "a.b/1"}))
	assert.Equal(t, "a.b", anchorLabel(&llx.AssetValue{ResourceType: "a.b"}))
	assert.Equal(t, "<none>", anchorLabel(nil))
}

// The full host-side path: reverse edge -> a runtime for the target -> its root
// resource, pinned to the runtime that answers it. The coordinator is mocked so
// this stays a unit test; connecting a real recorded asset through the mock
// provider is exercised by the integration suite.
func TestResolveAssetRootBindsToTheTargetRuntime(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)

	host := &inventory.Asset{Name: "host", PlatformIds: []string{"//platformid/host"}}
	anchor := &llx.AssetValue{ResourceType: "claude.code.mcpServer", ResourceId: "claude.code.mcpServer/github"}

	r := runtimeWith(host, recordedAsset("mcp-github", anchoredOn(host, anchor.ResourceType, anchor.ResourceId)))
	r.coordinator = mockC

	// The sub-runtime arrives already connected, which is what RuntimeFor does
	// when the target deduped onto a runtime the coordinator already had.
	sub := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		Provider: &ConnectedProvider{
			Instance:   &RunningProvider{ID: "os", Name: "os"},
			Connection: &plugin.ConnectRes{Asset: &inventory.Asset{Name: "mcp-github"}},
		},
	}

	var connectedTo *inventory.Asset
	mockC.EXPECT().RuntimeFor(gomock.Any(), r).Times(1).
		DoAndReturn(func(asset *inventory.Asset, parent *Runtime) (*Runtime, error) {
			connectedTo = asset
			return sub, nil
		})
	// CreateResource resolves the root through the sub-runtime's schema. A
	// resource with no provider is a bridging resource, which needs no
	// connection - enough to prove the routing without starting a plugin.
	mockC.EXPECT().Schema().AnyTimes().Return(mockSchema)
	mockSchema.EXPECT().Lookup("mcp").AnyTimes().Return(&resources.ResourceInfo{
		Id: "mcp", Name: "mcp",
	})

	resource, err := r.ResolveAssetRoot(anchor, "mcp")
	require.NoError(t, err)
	require.NotNil(t, resource)

	assert.Equal(t, "mcp-github", connectedTo.Name, "resolved the asset named by the reverse edge")
	assert.Equal(t, "mcp", resource.MqlName())

	bound, ok := resource.(llx.RuntimeBoundResource)
	require.True(t, ok, "the resource has to name the runtime that answers it")
	assert.Same(t, sub, bound.MqlRuntime())

	// A second resolve of the same anchor reuses the runtime rather than
	// connecting again: a discovered asset has no mrn or platform ids yet, so
	// coordinator dedupe cannot see that these are the same asset.
	again, err := r.ResolveAssetRoot(anchor, "mcp")
	require.NoError(t, err)
	assert.Same(t, sub, again.(llx.RuntimeBoundResource).MqlRuntime())

	// And the parent owns it: closing the parent closes what it opened.
	assert.Equal(t, []*Runtime{sub}, r.subRuntimes)
}

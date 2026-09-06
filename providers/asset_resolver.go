// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/recording"
)

// Cross-asset resolution (ADR 031). A query reaches another asset through a
// typed `asset<root>` value, and llx hands it here because llx cannot connect:
// a builtin sees only its runtime, which has no coordinator. This file is the
// host side of that - find the target asset, get a runtime for it, hand back
// its root resource.
//
// The backing is not a second mechanism. Recorded and live resolution are the
// same `RuntimeFor` + `Connect` path with a different connection on the target
// asset; this phase builds the recorded one, where the target carries a `mock`
// connection and answers from the recording the parent already holds.

// maxAssetResolveDepth bounds a chain of resolutions. A per-runtime cache stops
// a single asset being connected twice, but not an A -> B -> A cycle across
// runtimes, because each hop is a legitimate cache miss on a different runtime.
const maxAssetResolveDepth = 5

// mockConnectLock serializes the recording handoff and connect for a resolved
// asset. See the comment at its use.
var mockConnectLock sync.Mutex

// ResolveAssetRoot implements llx.AssetResolver.
func (r *Runtime) ResolveAssetRoot(v *llx.AssetValue, root string) (llx.Resource, error) {
	if v == nil {
		return nil, errors.New("cannot resolve an empty asset reference")
	}
	if root == "" {
		return nil, errors.New("cannot resolve asset " + anchorLabel(v) + ": no root resource requested")
	}

	sub, err := r.runtimeForAnchor(v)
	if err != nil {
		return nil, err
	}

	resource, err := sub.CreateResource(root, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read "+root+" on asset "+anchorLabel(v))
	}

	// The resource carries the runtime that answers it, so every field read
	// above this point routes to the target asset without llx knowing anything
	// about assets.
	return llx.BindResource(resource, sub), nil
}

// runtimeForAnchor returns the runtime for the asset an anchor points at,
// connecting it on first use.
//
// The cache is keyed on the anchor rather than on asset identity because a
// discovered asset has neither an MRN nor platform ids until its provider's
// Connect assigns them - so coordinator dedupe (coordinator.go:191) cannot see
// that two resolutions mean the same asset, and `mcpServers.map(running.tools)`
// would open one connection per server without this.
func (r *Runtime) runtimeForAnchor(v *llx.AssetValue) (*Runtime, error) {
	key := v.ResourceType + "\x00" + v.ResourceId

	r.mu.Lock()
	if sub, ok := r.resolvedAssets[key]; ok {
		r.mu.Unlock()
		return sub, nil
	}
	chain := r.resolveChain
	r.mu.Unlock()

	if len(chain) >= maxAssetResolveDepth {
		return nil, errors.New("cannot resolve asset " + anchorLabel(v) + ": resolution nested too deeply (" +
			strings.Join(append(chain, anchorLabel(v)), " -> ") + ")")
	}

	target, err := r.targetAssetForAnchor(v)
	if err != nil {
		return nil, err
	}

	sub, err := r.coordinator.RuntimeFor(target, r)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create a runtime for asset "+anchorLabel(v))
	}

	// RuntimeFor may hand back an already-connected runtime when the target
	// deduped onto one the coordinator knows.
	if sub.Provider == nil || sub.Provider.Connection == nil {
		// SetRecording after the provider is selected and before connecting,
		// as discovery does (discovery.go:71). It is not just bookkeeping: it
		// is what hands the builtin mock service the runtime whose recording it
		// reads, and without it Connect dereferences a nil one.
		//
		// The builtin services are singletons (runtime.go:963), so that handoff
		// is last-writer-wins across runtimes. Serializing connect keeps a
		// concurrent resolution - `servers.map(running.tools)` - from connecting
		// one asset against another's recording.
		mockConnectLock.Lock()
		if err = sub.SetRecording(r.Recording()); err == nil {
			err = sub.Connect(&plugin.ConnectReq{
				Asset:    target,
				Features: r.features,
				Upstream: r.UpstreamConfig,
			})
		}
		mockConnectLock.Unlock()
		if err != nil {
			sub.Close()
			return nil, errors.Wrap(err, "cannot connect to asset "+anchorLabel(v))
		}
	}

	sub.mu.Lock()
	sub.resolveChain = append(append([]string{}, chain...), anchorLabel(v))
	sub.mu.Unlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	// Another goroutine may have resolved the same anchor while this one was
	// connecting. Keep the first and drop ours rather than leaking it.
	if existing, ok := r.resolvedAssets[key]; ok {
		go sub.Close()
		return existing, nil
	}
	if r.resolvedAssets == nil {
		r.resolvedAssets = map[string]*Runtime{}
	}
	r.resolvedAssets[key] = sub
	// Sub-runtimes are owned by the parent and torn down with it, rather than
	// left for the coordinator to reap.
	r.subRuntimes = append(r.subRuntimes, sub)
	return sub, nil
}

// targetAssetForAnchor finds the asset an anchor points at and builds the stub
// to connect it with.
//
// The lookup runs over the **reverse edge**: discovery writes an
// inventory.AssetRelationship on the discovered asset naming the resource on
// its parent that anchors it (ADR 030), and this is the first consumer of that
// edge at runtime. Nothing about the parent's own tree is consulted, which is
// what keeps forward and reverse in agreement by construction.
func (r *Runtime) targetAssetForAnchor(v *llx.AssetValue) (*inventory.Asset, error) {
	multi, ok := r.Recording().(recording.MultiAsset)
	if !ok {
		return nil, errors.New("cannot resolve asset " + anchorLabel(v) +
			": no recording to resolve it from, and live connect is not implemented yet")
	}

	self := r.asset()
	var matches []*inventory.Asset
	var exact *inventory.Asset
	for _, candidate := range multi.GetAssetRecordings() {
		if candidate == nil || candidate.Asset == nil {
			continue
		}
		for _, rel := range candidate.Asset.Relationships {
			if rel == nil || rel.ResourceType != v.ResourceType || rel.ResourceId != v.ResourceId {
				continue
			}
			matches = append(matches, candidate.Asset)
			if self != nil && rel.Asset != nil && sameAsset(rel.Asset, self) {
				exact = candidate.Asset
			}
			break
		}
	}

	// An anchor id is only unique within one parent, so several recorded assets
	// can carry the same anchor when the recording holds several hosts. The edge
	// that names *this* asset as the parent is the right one.
	target := exact
	if target == nil {
		switch len(matches) {
		case 0:
			return nil, errors.New("cannot resolve asset " + anchorLabel(v) +
				": no recorded asset is anchored on it")
		case 1:
			target = matches[0]
		default:
			return nil, errors.New("cannot resolve asset " + anchorLabel(v) +
				": several recorded assets are anchored on it and none names this asset as its parent")
		}
	}

	// A mock connection is what makes this the recorded leg: the mock provider
	// looks the asset up in the recording the parent already holds and mock-
	// connects the provider that actually owns it. The live leg is the same
	// call with the target's real connection instead.
	return &inventory.Asset{
		Name:        target.Name,
		Mrn:         target.Mrn,
		PlatformIds: target.PlatformIds,
		Connections: []*inventory.Config{{
			Type: "mock",
		}},
	}, nil
}

// closeSubRuntimes tears down the runtimes this one opened for other assets.
func (r *Runtime) closeSubRuntimes() {
	r.mu.Lock()
	subs := r.subRuntimes
	r.subRuntimes = nil
	r.resolvedAssets = nil
	r.mu.Unlock()

	for _, sub := range subs {
		if sub == nil || sub == r {
			continue
		}
		log.Debug().Msg("closing a runtime opened for cross-asset resolution")
		sub.Close()
	}
}

// sameAsset reports whether two asset references describe the same asset, using
// the identifiers the coordinator dedupes on (coordinator.go:191) and falling
// back to the name for a stub that carries neither.
func sameAsset(a *inventory.Asset, b *inventory.Asset) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Mrn != "" && a.Mrn == b.Mrn {
		return true
	}
	for _, id := range a.PlatformIds {
		for _, other := range b.PlatformIds {
			if id == other {
				return true
			}
		}
	}
	if a.Mrn != "" || b.Mrn != "" || len(a.PlatformIds) > 0 || len(b.PlatformIds) > 0 {
		// Both sides carry a real identifier and they did not match. Falling
		// through to the name here would let two different assets that happen
		// to share a display name look like one.
		return false
	}
	return a.Name != "" && a.Name == b.Name
}

// anchorLabel names an anchor for an error message.
func anchorLabel(v *llx.AssetValue) string {
	if v == nil {
		return "<none>"
	}
	if v.ResourceId == "" {
		return v.ResourceType
	}
	return v.ResourceType + " (" + v.ResourceId + ")"
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/recording"
)

// newMultiAssetTestRecording builds a recording with two assets, each reachable
// by its own connection id (1 and 2), mirroring how a multi-asset recording
// file is indexed when loaded from disk.
func newMultiAssetTestRecording(t *testing.T) llx.Recording {
	t.Helper()

	asset1 := &inventory.Asset{
		Name:        "asset1",
		Mrn:         "asset1-mrn",
		PlatformIds: []string{"//platformid.api.mondoo.app/hostname/asset1"},
		Platform:    &inventory.Platform{Name: "debian"},
	}
	asset2 := &inventory.Asset{
		Name:        "asset2",
		Mrn:         "asset2-mrn",
		PlatformIds: []string{"//platformid.api.mondoo.app/hostname/asset2"},
		Platform:    &inventory.Platform{Name: "debian"},
	}

	rec := recording.NewWithAsset(asset1)
	rec.EnsureAsset(asset1, "recording", 1, &inventory.Config{Type: "recording", Id: 1})
	rec.EnsureAsset(asset2, "recording", 2, &inventory.Config{Type: "recording", Id: 2})
	return rec
}

func storeUsersList(connID uint32, value string) *plugin.StoreReq {
	return &plugin.StoreReq{
		Connection: connID,
		Resources: []*plugin.ResourceData{{
			Name: "users",
			Id:   "qrid-derived-clone-id",
			Fields: map[string]*llx.Result{
				"list": {Data: llx.StringPrimitive(value)},
			},
		}},
	}
}

// primitiveString extracts a string from a primitive that round-tripped
// through the recording (string values come back as raw bytes).
func primitiveString(p *llx.Primitive) string {
	return string(p.Value)
}

func getUsersList(connID uint32) *plugin.DataReq {
	return &plugin.DataReq{
		Connection: connID,
		Resource:   "users",
		ResourceId: "qrid-derived-clone-id",
		Field:      "list",
	}
}

// TestRecordingProviderConnScopedRouting reproduces
// https://github.com/mondoohq/mql/issues/9461: a single static recording
// provider serves all assets of a multi-asset recording. When queries store
// computed data (e.g. `where` rewrap clones, whose resource id is derived from
// the query checksum and is therefore identical across assets), the provider
// must route GetData/StoreData by the request's connection. Routing by the
// shared selectedAsset instead makes one asset's data land in (or get read
// from) another asset's recording section.
func TestRecordingProviderConnScopedRouting(t *testing.T) {
	rec := newMultiAssetTestRecording(t)

	// Simulate the shared singleton after the scan framework connected asset2
	// last: selectedAsset points at asset2 while both assets are being scanned.
	asset2 := &inventory.Asset{
		Mrn:         "asset2-mrn",
		PlatformIds: []string{"//platformid.api.mondoo.app/hostname/asset2"},
		Platform:    &inventory.Platform{Name: "debian"},
		Connections: []*inventory.Config{{Type: "recording", Id: 2}},
	}
	p := NewRecordingProvider(WithRecording(rec), WithAsset(asset2))

	// Both assets store a computed clone under the same (checksum-derived)
	// resource id, each tagged with its own connection.
	_, err := p.StoreData(storeUsersList(1, "asset1-list"))
	require.NoError(t, err)
	_, err = p.StoreData(storeUsersList(2, "asset2-list"))
	require.NoError(t, err)

	// Each connection must read back its own asset's data.
	res, err := p.GetData(getUsersList(1))
	require.NoError(t, err)
	assert.Equal(t, "asset1-list", primitiveString(res.Data))

	res, err = p.GetData(getUsersList(2))
	require.NoError(t, err)
	assert.Equal(t, "asset2-list", primitiveString(res.Data))

	// A lookup for data that was never stored under a connection must not be
	// served from another asset's section.
	_, err = p.GetData(&plugin.DataReq{
		Connection: 2,
		Resource:   "users",
		ResourceId: "only-stored-on-conn-1",
		Field:      "list",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't exist")

	_, err = p.StoreData(&plugin.StoreReq{
		Connection: 1,
		Resources: []*plugin.ResourceData{{
			Name:   "users",
			Id:     "only-stored-on-conn-1",
			Fields: map[string]*llx.Result{"list": {Data: llx.StringPrimitive("x")}},
		}},
	})
	require.NoError(t, err)

	_, err = p.GetData(&plugin.DataReq{
		Connection: 2,
		Resource:   "users",
		ResourceId: "only-stored-on-conn-1",
		Field:      "list",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't exist")
}

// TestRecordingProviderSelectedAssetFallback ensures the legacy programmatic
// flow (no connection set on requests) still resolves via the selected asset.
func TestRecordingProviderSelectedAssetFallback(t *testing.T) {
	rec := newMultiAssetTestRecording(t)

	asset1 := &inventory.Asset{
		Mrn:         "asset1-mrn",
		PlatformIds: []string{"//platformid.api.mondoo.app/hostname/asset1"},
		Platform:    &inventory.Platform{Name: "debian"},
		Connections: []*inventory.Config{{Type: "recording", Id: 1}},
	}
	p := NewRecordingProvider(WithRecording(rec), WithAsset(asset1))

	_, err := p.StoreData(storeUsersList(0, "legacy-value"))
	require.NoError(t, err)

	res, err := p.GetData(getUsersList(0))
	require.NoError(t, err)
	assert.Equal(t, "legacy-value", primitiveString(res.Data))

	// The data must have landed in the selected asset's section (conn id 1),
	// and must not be visible from the other asset's connection.
	res, err = p.GetData(getUsersList(1))
	require.NoError(t, err)
	assert.Equal(t, "legacy-value", primitiveString(res.Data))

	_, err = p.GetData(getUsersList(2))
	require.Error(t, err)
}

// TestRecordingProviderStoreWithoutAsset keeps the existing error behavior for
// StoreData when neither a connection nor a selected asset is available.
func TestRecordingProviderStoreWithoutAsset(t *testing.T) {
	p := NewRecordingProvider(WithRecording(newMultiAssetTestRecording(t)))

	_, err := p.StoreData(storeUsersList(0, "x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no asset selected")
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

type readOnly struct {
	*recording
}

func (n *readOnly) Save() error {
	return nil
}

func (n *readOnly) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
	// For read-only recordings we are still loading from file, so that means
	// we are severely lacking connection IDs.
	lookup := llx.AssetRecordingLookup{
		Mrn:         asset.Mrn,
		PlatformIds: asset.PlatformIds,
	}
	if existing, ok := n.resolveAsset(lookup); ok {
		n.assets.Set(connIdKey(connectionID), existing)
	}
}

func (n *readOnly) AddData(req llx.AddDataReq) {
}

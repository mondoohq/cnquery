// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"fmt"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

var _ llx.Recording = &readOnly{}

type readOnly struct {
	*recording
}

func (n *readOnly) Save() error {
	return nil
}

func (n *readOnly) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
	// For read-only recordings we are still loading from file, so that means
	// we are severely lacking connection IDs.
	existing := n.getExistingAsset(asset)
	if existing != nil {
		n.assets.Set(fmt.Sprintf("%d", connectionID), existing)
	}
}

func (n *readOnly) AddData(req llx.AddDataReq) {
}

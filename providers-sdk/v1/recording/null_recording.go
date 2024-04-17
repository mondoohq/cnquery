// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

type Null struct{}

func (n Null) Save() error {
	return nil
}

func (n Null) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
}

func (n Null) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
}

func (n Null) GetData(connectionID uint32, resource string, id string, field string) (*llx.RawData, bool) {
	return nil, false
}

func (n Null) GetResource(connectionID uint32, resource string, id string) (map[string]*llx.RawData, bool) {
	return nil, false
}

func (n Null) GetAssetData(assetMrn string) (map[string]*llx.ResourceRecording, bool) {
	return nil, false
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
)

type Runtime interface {
	AssetMRN() string
	Unregister(watcherUID string) error
	CreateResource(name string, args map[string]*Primitive) (Resource, error)
	CloneResource(src Resource, id string, fields []string, args map[string]*Primitive) (Resource, error)
	WatchAndUpdate(resource Resource, field string, watcherUID string, callback func(res interface{}, err error)) error
	Schema() resources.ResourcesSchema
	Close()

	// Recording handlers
	Recording() Recording
	SetRecording(recording Recording) error
	AssetUpdated(asset *inventory.Asset)
}

type Recording interface {
	Save() error
	EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config)
	AddData(connectionID uint32, resource string, id string, field string, data *RawData)
	GetData(connectionID uint32, resource string, id string, field string) (*RawData, bool)
	GetResource(connectionID uint32, resource string, id string) (map[string]*RawData, bool)
	GetAssetData(assetMrn string) (map[string]*ResourceRecording, bool)
}

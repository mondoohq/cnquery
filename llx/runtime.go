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
	WatchAndUpdate(resource Resource, field string, watcherUID string, callback func(res any, err error)) error
	Schema() resources.ResourcesSchema
	Close()

	// Recording handlers
	Recording() Recording
	SetRecording(recording Recording) error
	AssetUpdated(asset *inventory.Asset)
}

// Allows looking up data for assets, based on different asset identifiers.
// If set, Mrn is preferred, followed by PlatformIds, and lastly ConnectionId.
type AssetRecordingLookup struct {
	ConnectionId uint32
	Mrn          string
	PlatformIds  []string
}

type AddDataReq struct {
	// the id of the connection that was used to fetch the data
	ConnectionID uint32
	// the resource type name
	Resource string
	// the id of the resource as returned by the connection
	ResourceID string
	// the resource field, if specified
	Field string
	// the resource data
	Data *RawData
	// the id of the resource as requested towards the connection
	RequestResourceId string
}

type Recording interface {
	Save() error
	EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config)
	AddData(req AddDataReq)
	GetData(lookup AssetRecordingLookup, resource string, resourceId string, field string) (*RawData, bool)
	GetResource(lookup AssetRecordingLookup, resource string, resourceId string) (map[string]*RawData, bool)
	GetAssetData(assetMrn string) (map[string]*ResourceRecording, bool)
	GetAssets() []*inventory.Asset
}

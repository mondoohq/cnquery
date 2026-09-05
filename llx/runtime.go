// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
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

// AssetRootSource reports the resource that roots the connected asset's tree
// (ADR 031). It is an optional interface, checked by mqlc.NewConfigFrom, so a
// runtime that has no notion of a connection - a mock, an embedder's own - is
// unaffected.
//
// It is read at compile time, which is why the root is a static declaration on
// the provider rather than something the connection answers after connecting:
// content compiled where nothing is connected (a policy bundle built upstream)
// still has to compile, and there `_` degrades to the error it always was.
type AssetRootSource interface {
	// AssetRoot returns the root resource name the query should be bounded by,
	// or "" when the connected provider declares none. This is what the
	// connection reported, so on a Linux host it is the Linux root and not the
	// provider's catch-all.
	AssetRoot() string
	// DeclaredAssetRoot returns the root the provider declares statically. For a
	// provider serving several platforms that is the union of its roots, which
	// is a compile-time receiver rather than a platform, so it is worth telling
	// apart from the effective root: a diagnostic that offers the union as an
	// alternative platform is offering nothing.
	DeclaredAssetRoot() string
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
	// optionally identifies the asset this data belongs to. Preferred over
	// ConnectionID when it carries an Mrn or platform ids, because connection
	// ids in a recording that was loaded from disk are not guaranteed to be
	// unique across its assets.
	Lookup AssetRecordingLookup
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

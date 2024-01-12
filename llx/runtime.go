// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import "go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"

type Runtime interface {
	AssetMRN() string
	Unregister(watcherUID string) error
	CreateResource(name string, args map[string]*Primitive) (Resource, error)
	CloneResource(src Resource, id string, fields []string, args map[string]*Primitive) (Resource, error)
	WatchAndUpdate(resource Resource, field string, watcherUID string, callback func(res interface{}, err error)) error
	Schema() Schema
	Close()
}

type Schema interface {
	Lookup(resource string) *resources.ResourceInfo
	LookupField(resource string, field string) (*resources.ResourceInfo, *resources.Field)
	AllResources() map[string]*resources.ResourceInfo
}

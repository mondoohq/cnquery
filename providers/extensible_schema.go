// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"golang.org/x/exp/slices"
)

type extensibleSchema struct {
	aggregate      resources.Schema
	prioritization []string

	loaded        map[string]*resources.Schema
	runtime       *Runtime
	lastRefreshed int64
	lockAll       sync.Mutex // only used in getting all schemas
	lockAdd       sync.Mutex // only used when adding a schema
}

func newExtensibleSchema() extensibleSchema {
	return extensibleSchema{
		aggregate: resources.Schema{
			Resources: map[string]*resources.ResourceInfo{},
		},
		loaded:         map[string]*resources.Schema{},
		prioritization: []string{BuiltinCoreID},
	}
}

func (x *extensibleSchema) loadAllSchemas() {
	x.lockAll.Lock()
	defer x.lockAll.Unlock()

	// If another goroutine started to load this before us, it will be locked until
	// we complete to load everything and then it will be dumped into this
	// position. At this point, if it has been loaded we can return safely, since
	// we don't unlock until we are finished loading.
	if x.lastRefreshed >= LastProviderInstall {
		return
	}
	x.lastRefreshed = LastProviderInstall

	providers, err := ListActive()
	if err != nil {
		log.Error().Err(err).Msg("failed to list all providers, can't load additional schemas")
		return
	}

	for name := range providers {
		schema, err := x.runtime.coordinator.LoadSchema(name)
		if err != nil {
			log.Error().Err(err).Msg("load schema failed")
		} else {
			x.Add(name, schema)
		}
	}
}

func (x *extensibleSchema) Close() {
	x.loaded = map[string]*resources.Schema{}
	x.aggregate.Resources = map[string]*resources.ResourceInfo{}
}

func (x *extensibleSchema) Lookup(name string) *resources.ResourceInfo {
	if found, ok := x.aggregate.Resources[name]; ok {
		return found
	}
	if x.lastRefreshed >= LastProviderInstall {
		return nil
	}

	x.loadAllSchemas()
	x.refresh()

	return x.aggregate.Resources[name]
}

func (x *extensibleSchema) LookupField(resource string, field string) (*resources.ResourceInfo, *resources.Field) {
	found, ok := x.aggregate.Resources[resource]
	if !ok {
		if x.lastRefreshed >= LastProviderInstall {
			return nil, nil
		}

		x.loadAllSchemas()
		x.refresh()

		found, ok = x.aggregate.Resources[resource]
		if !ok {
			return nil, nil
		}
		return found, found.Fields[field]
	}

	fieldObj, ok := found.Fields[field]
	if ok {
		return found, fieldObj
	}
	if x.lastRefreshed >= LastProviderInstall {
		return found, nil
	}

	x.loadAllSchemas()
	x.refresh()

	return found, found.Fields[field]
}

func (x *extensibleSchema) Add(name string, schema *resources.Schema) {
	if schema == nil {
		return
	}
	if name == "" {
		log.Error().Msg("tried to add a schema with no name")
		return
	}

	x.lockAdd.Lock()
	x.loaded[name] = schema
	x.lockAdd.Unlock()
}

// Prioritize the provider IDs in the order that is provided. Any other
// provider comes later and in any random order.
func (x *extensibleSchema) prioritizeIDs(prioritization ...string) {
	x.prioritization = prioritization
}

func (x *extensibleSchema) refresh() {
	x.lockAll.Lock()
	defer x.lockAll.Unlock()

	res := resources.Schema{
		Resources: map[string]*resources.ResourceInfo{},
	}
	for id, schema := range x.loaded {
		if !slices.Contains(x.prioritization, id) {
			res.Add(schema)
		}
	}

	for i := len(x.prioritization) - 1; i >= 0; i-- {
		id := x.prioritization[i]
		if s := x.loaded[id]; s != nil {
			res.Add(s)
		}
	}
	x.aggregate.Resources = res.Resources
}

func (x *extensibleSchema) Schema() *resources.Schema {
	if x.aggregate.Resources == nil {
		x.refresh()
	}
	return &x.aggregate
}

func (x *extensibleSchema) AllResources() map[string]*resources.ResourceInfo {
	if x.lastRefreshed < LastProviderInstall {
		x.loadAllSchemas()
		x.refresh()
	} else if x.aggregate.Resources == nil {
		x.refresh()
	}

	return x.aggregate.Resources
}

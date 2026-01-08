// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
)

// FilteredSchema wraps a ResourcesSchema and filters resources to only show
// those from connected providers or cross-provider resources (like core).
type FilteredSchema struct {
	schema             resources.ResourcesSchema
	connectedProviders map[string]struct{}
}

// Providers that are always available regardless of connection
var alwaysAvailableProviders = []string{
	"go.mondoo.com/cnquery/v9/providers/core",
	"go.mondoo.com/cnquery/v9/providers/network",
}

// NewFilteredSchema creates a new FilteredSchema that only exposes resources
// from the specified providers. The core and network providers are always included
// as they are available regardless of the connection type.
func NewFilteredSchema(schema resources.ResourcesSchema, providerIDs []string) *FilteredSchema {
	providers := make(map[string]struct{}, len(providerIDs)+len(alwaysAvailableProviders))
	for _, id := range providerIDs {
		providers[id] = struct{}{}
	}
	// Always include core and network providers
	for _, id := range alwaysAvailableProviders {
		providers[id] = struct{}{}
	}

	return &FilteredSchema{
		schema:             schema,
		connectedProviders: providers,
	}
}

// Lookup returns the resource info for a given resource name.
// It returns nil if the resource is not from a connected provider.
func (f *FilteredSchema) Lookup(resource string) *resources.ResourceInfo {
	info := f.schema.Lookup(resource)
	if info == nil {
		return nil
	}
	if !f.isProviderConnected(info.Provider) {
		return nil
	}
	return info
}

// LookupField returns the resource info and field for a given resource and field name.
func (f *FilteredSchema) LookupField(resource string, field string) (*resources.ResourceInfo, *resources.Field) {
	info, fieldInfo := f.schema.LookupField(resource, field)
	if info == nil {
		return nil, nil
	}
	if !f.isProviderConnected(info.Provider) {
		return nil, nil
	}
	return info, fieldInfo
}

// FindField finds a field in a resource, including embedded fields.
func (f *FilteredSchema) FindField(resource *resources.ResourceInfo, field string) (resources.FieldPath, []*resources.Field, bool) {
	return f.schema.FindField(resource, field)
}

// AllResources returns only resources from connected providers.
func (f *FilteredSchema) AllResources() map[string]*resources.ResourceInfo {
	all := f.schema.AllResources()
	filtered := make(map[string]*resources.ResourceInfo, len(all))

	for name, info := range all {
		if f.isProviderConnected(info.Provider) {
			filtered[name] = info
		}
	}

	return filtered
}

// AllDependencies returns all provider dependencies.
func (f *FilteredSchema) AllDependencies() map[string]*resources.ProviderInfo {
	return f.schema.AllDependencies()
}

// isProviderConnected checks if a provider is in the connected providers set.
// Empty provider string means cross-provider resource, which is always included.
func (f *FilteredSchema) isProviderConnected(provider string) bool {
	if provider == "" {
		return true
	}
	_, ok := f.connectedProviders[provider]
	return ok
}

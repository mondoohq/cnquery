// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"sort"

	"go.mondoo.com/cnquery/v10/llx"
)

type Asset struct {
	Asset       assetInfo    `json:"asset"`
	Connections []connection `json:"connections"`
	Resources   []Resource   `json:"resources"`

	connections map[string]*connection `json:"-"`
	resources   map[string]*Resource   `json:"-"`
}

type assetInfo struct {
	ID          string            `json:"id"`
	PlatformIDs []string          `json:"platformIDs,omitempty"`
	Name        string            `json:"name,omitempty"`
	Arch        string            `json:"arch,omitempty"`
	Title       string            `json:"title,omitempty"`
	Family      []string          `json:"family,omitempty"`
	Build       string            `json:"build,omitempty"`
	Version     string            `json:"version,omitempty"`
	Kind        string            `json:"kind,omitempty"`
	Runtime     string            `json:"runtime,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type connection struct {
	Url        string `json:"url"`
	ProviderID string `json:"provider"`
	Connector  string `json:"connector"`
	Version    string `json:"version"`
	id         uint32 `json:"-"`
}

type Resource struct {
	Resource string
	ID       string
	Fields   map[string]*llx.RawData
}

func (asset *Asset) finalize() {
	asset.Resources = make([]Resource, len(asset.resources))
	asset.Connections = make([]connection, len(asset.connections))

	i := 0
	for _, v := range asset.resources {
		asset.Resources[i] = *v
		i++
	}

	sort.Slice(asset.Resources, func(i, j int) bool {
		a := asset.Resources[i]
		b := asset.Resources[j]
		if a.Resource == b.Resource {
			return a.ID < b.ID
		}
		return a.Resource < b.Resource
	})

	i = 0
	for _, v := range asset.connections {
		asset.Connections[i] = *v
		i++
	}
}

func (asset *Asset) GetResource(name string, id string) (*Resource, bool) {
	r, ok := asset.resources[name+"\x00"+id]
	return r, ok
}

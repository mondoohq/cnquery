// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"sort"

	"go.mondoo.com/cnquery/v12/llx"
)

type ResourceIdLookup struct {
	// The resource id as requested towards the connection
	RequestId string `json:"requestId"`
	// The resource id as returned by the connection
	ResponseId string `json:"responseId"`
}

type Asset struct {
	Asset       *assetInfo   `json:"asset"`
	Connections []connection `json:"connections"`
	Resources   []Resource   `json:"resources"`
	// A lookup of requested resources to their actual ID.
	// This is required to resolve cases where a resource is requested by one ID (usually empty ID)
	// and the connection responds with another (resolved) ID. This mapping allows us to mimic
	// the same behavior when reading/replaying recordings.
	//
	// The key is the resource name + request ID, e.g.
	// "aws.ec2.instance\x00123": "i-1234567890abcdef0"
	// "azure.subscription\x001": "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	IdsLookup map[string]string `json:"idsLookup,omitempty"`

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
	Annotations map[string]string `json:"annotations,omitempty"`
}

type connection struct {
	Url        string `json:"url"`
	ProviderID string `json:"provider"`
	Connector  string `json:"connector"`
	Version    string `json:"version"`
	Id         uint32 `json:"id"`
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

func (asset *Asset) RefreshCache() {
	asset.resources = make(map[string]*Resource, len(asset.Resources))
	asset.connections = make(map[string]*connection, len(asset.Connections))

	for _, resource := range asset.Resources {
		asset.resources[resource.Resource+"\x00"+resource.ID] = &resource
	}

	for _, conn := range asset.Connections {
		asset.connections[conn.Url] = &conn
	}
}

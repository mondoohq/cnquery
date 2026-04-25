// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerDatacenterInternal struct {
	cacheLocation                      *hcloud.Location
	cacheSupportedServerTypes          []*hcloud.ServerType
	cacheAvailableServerTypes          []*hcloud.ServerType
	cacheServerTypesAvailableMigration []*hcloud.ServerType
}

func (r *mqlHetznerDatacenter) id() (string, error) {
	return fmt.Sprintf("hetzner.datacenter/%d", r.Id.Data), nil
}

func (h *mqlHetzner) datacenters() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Datacenter, *hcloud.Response, error) {
		return c.Client().Datacenter.List(ctx(), hcloud.DatacenterListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, dc := range items {
		res, err := newMqlHetznerDatacenter(h.MqlRuntime, dc)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerDatacenter(runtime *plugin.Runtime, dc *hcloud.Datacenter) (*mqlHetznerDatacenter, error) {
	res, err := CreateResource(runtime, "hetzner.datacenter", map[string]*llx.RawData{
		"__id":        llx.StringData(fmt.Sprintf("hetzner.datacenter/%d", dc.ID)),
		"id":          llx.IntData(dc.ID),
		"name":        llx.StringData(dc.Name),
		"description": llx.StringData(dc.Description),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerDatacenter)
	m.cacheLocation = dc.Location
	m.cacheSupportedServerTypes = dc.ServerTypes.Supported
	m.cacheAvailableServerTypes = dc.ServerTypes.Available
	m.cacheServerTypesAvailableMigration = dc.ServerTypes.AvailableForMigration
	return m, nil
}

func initHetznerDatacenter(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	dc, _, err := conn(runtime).Client().Datacenter.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if dc == nil {
		return nil, nil, notFoundErr("datacenter", id)
	}
	res, err := newMqlHetznerDatacenter(runtime, dc)
	return args, res, err
}

func (m *mqlHetznerDatacenter) location() (*mqlHetznerLocation, error) {
	return resolveTypedResource(&m.Location, m.cacheLocation, func(loc *hcloud.Location) (*mqlHetznerLocation, error) {
		return newMqlHetznerLocation(m.MqlRuntime, loc)
	})
}

func (m *mqlHetznerDatacenter) supportedServerTypes() ([]any, error) {
	return serverTypeRefs(m.MqlRuntime, m.cacheSupportedServerTypes)
}

func (m *mqlHetznerDatacenter) availableServerTypes() ([]any, error) {
	return serverTypeRefs(m.MqlRuntime, m.cacheAvailableServerTypes)
}

func (m *mqlHetznerDatacenter) serverTypesAvailableForMigration() ([]any, error) {
	return serverTypeRefs(m.MqlRuntime, m.cacheServerTypesAvailableMigration)
}

func serverTypeRefs(runtime *plugin.Runtime, types []*hcloud.ServerType) ([]any, error) {
	out := make([]any, 0, len(types))
	for _, t := range types {
		ref, err := NewResource(runtime, "hetzner.serverType", map[string]*llx.RawData{
			"id": llx.IntData(t.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

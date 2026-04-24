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
	cacheLocation *hcloud.Location
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
	st := map[string]any{}
	if dc.ServerTypes.Supported != nil {
		ids := make([]any, 0, len(dc.ServerTypes.Supported))
		for _, t := range dc.ServerTypes.Supported {
			ids = append(ids, t.ID)
		}
		st["supported"] = ids
	}
	if dc.ServerTypes.Available != nil {
		ids := make([]any, 0, len(dc.ServerTypes.Available))
		for _, t := range dc.ServerTypes.Available {
			ids = append(ids, t.ID)
		}
		st["available"] = ids
	}
	if dc.ServerTypes.AvailableForMigration != nil {
		ids := make([]any, 0, len(dc.ServerTypes.AvailableForMigration))
		for _, t := range dc.ServerTypes.AvailableForMigration {
			ids = append(ids, t.ID)
		}
		st["availableForMigration"] = ids
	}

	res, err := CreateResource(runtime, "hetzner.datacenter", map[string]*llx.RawData{
		"__id":        llx.StringData(fmt.Sprintf("hetzner.datacenter/%d", dc.ID)),
		"id":          llx.IntData(dc.ID),
		"name":        llx.StringData(dc.Name),
		"description": llx.StringData(dc.Description),
		"serverTypes": llx.DictData(st),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerDatacenter)
	m.cacheLocation = dc.Location
	return m, nil
}

func initHetznerDatacenter(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return nil, nil, errIDRequired("datacenter")
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

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlHetznerLocation) id() (string, error) {
	return fmt.Sprintf("hetzner.location/%d", r.Id.Data), nil
}

func (h *mqlHetzner) locations() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Location, *hcloud.Response, error) {
		return c.Client().Location.List(ctx(), hcloud.LocationListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, loc := range items {
		res, err := newMqlHetznerLocation(h.MqlRuntime, loc)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerLocation(runtime *plugin.Runtime, loc *hcloud.Location) (*mqlHetznerLocation, error) {
	res, err := CreateResource(runtime, "hetzner.location", map[string]*llx.RawData{
		"__id":        llx.StringData(fmt.Sprintf("hetzner.location/%d", loc.ID)),
		"id":          llx.IntData(loc.ID),
		"name":        llx.StringData(loc.Name),
		"description": llx.StringData(loc.Description),
		"country":     llx.StringData(loc.Country),
		"city":        llx.StringData(loc.City),
		"latitude":    llx.FloatData(loc.Latitude),
		"longitude":   llx.FloatData(loc.Longitude),
		"networkZone": llx.StringData(string(loc.NetworkZone)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerLocation), nil
}

func initHetznerLocation(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	loc, _, err := conn(runtime).Client().Location.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if loc == nil {
		return nil, nil, notFoundErr("location", id)
	}
	res, err := newMqlHetznerLocation(runtime, loc)
	return args, res, err
}

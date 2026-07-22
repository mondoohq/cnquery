// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlHetznerIso) id() (string, error) {
	return fmt.Sprintf("hetzner.iso/%d", r.Id.Data), nil
}

func (h *mqlHetzner) isos() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.ISO, *hcloud.Response, error) {
		return c.Client().ISO.List(ctx(), hcloud.ISOListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, iso := range items {
		res, err := newMqlHetznerIso(h.MqlRuntime, iso)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerIso(runtime *plugin.Runtime, iso *hcloud.ISO) (*mqlHetznerIso, error) {
	dep := deprecationDict(iso.Deprecation)
	arch := ""
	if iso.Architecture != nil {
		arch = string(*iso.Architecture)
	}
	res, err := CreateResource(runtime, "hetzner.iso", map[string]*llx.RawData{
		"__id":         llx.StringData(fmt.Sprintf("hetzner.iso/%d", iso.ID)),
		"id":           llx.IntData(iso.ID),
		"name":         llx.StringData(iso.Name),
		"description":  llx.StringData(iso.Description),
		"type":         llx.StringData(string(iso.Type)),
		"deprecation":  llx.DictData(dep),
		"architecture": llx.StringData(arch),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerIso), nil
}

func initHetznerIso(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	iso, _, err := conn(runtime).Client().ISO.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if iso == nil {
		return nil, nil, notFoundErr("iso", id)
	}
	res, err := newMqlHetznerIso(runtime, iso)
	return args, res, err
}

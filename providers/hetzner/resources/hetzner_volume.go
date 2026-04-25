// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerVolumeInternal struct {
	cacheLocation *hcloud.Location
	cacheServerID int64
}

func (r *mqlHetznerVolume) id() (string, error) {
	return fmt.Sprintf("hetzner.volume/%d", r.Id.Data), nil
}

func (h *mqlHetzner) volumes() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Volume, *hcloud.Response, error) {
		return c.Client().Volume.List(ctx(), hcloud.VolumeListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, v := range items {
		res, err := newMqlHetznerVolume(h.MqlRuntime, v)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerVolume(runtime *plugin.Runtime, v *hcloud.Volume) (*mqlHetznerVolume, error) {
	format := ""
	if v.Format != nil {
		format = *v.Format
	}
	res, err := CreateResource(runtime, "hetzner.volume", map[string]*llx.RawData{
		"__id":        llx.StringData(fmt.Sprintf("hetzner.volume/%d", v.ID)),
		"id":          llx.IntData(v.ID),
		"name":        llx.StringData(v.Name),
		"size":        llx.IntData(int64(v.Size)),
		"status":      llx.StringData(string(v.Status)),
		"created":     llx.TimeDataPtr(timePtr(v.Created)),
		"linuxDevice": llx.StringData(v.LinuxDevice),
		"format":      llx.StringData(format),
		"protection":  llx.DictData(protectionDict(v.Protection.Delete)),
		"labels":      labelData(v.Labels),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerVolume)
	m.cacheLocation = v.Location
	if v.Server != nil {
		m.cacheServerID = v.Server.ID
	}
	return m, nil
}

func initHetznerVolume(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	v, _, err := conn(runtime).Client().Volume.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if v == nil {
		return nil, nil, notFoundErr("volume", id)
	}
	res, err := newMqlHetznerVolume(runtime, v)
	return args, res, err
}

func (m *mqlHetznerVolume) location() (*mqlHetznerLocation, error) {
	return resolveTypedResource(&m.Location, m.cacheLocation, func(loc *hcloud.Location) (*mqlHetznerLocation, error) {
		return newMqlHetznerLocation(m.MqlRuntime, loc)
	})
}

func (m *mqlHetznerVolume) server() (*mqlHetznerServer, error) {
	return serverRefByID(m.MqlRuntime, &m.Server, m.cacheServerID)
}

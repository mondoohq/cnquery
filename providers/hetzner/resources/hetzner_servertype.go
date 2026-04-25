// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerServerTypeInternal struct {
	cacheLocations []hcloud.ServerTypeLocation
}

func (r *mqlHetznerServerType) id() (string, error) {
	return fmt.Sprintf("hetzner.serverType/%d", r.Id.Data), nil
}

func (h *mqlHetzner) serverTypes() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.ServerType, *hcloud.Response, error) {
		return c.Client().ServerType.List(ctx(), hcloud.ServerTypeListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, t := range items {
		res, err := newMqlHetznerServerType(h.MqlRuntime, t)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerServerType(runtime *plugin.Runtime, t *hcloud.ServerType) (*mqlHetznerServerType, error) {
	res, err := CreateResource(runtime, "hetzner.serverType", map[string]*llx.RawData{
		"__id":         llx.StringData(fmt.Sprintf("hetzner.serverType/%d", t.ID)),
		"id":           llx.IntData(t.ID),
		"name":         llx.StringData(t.Name),
		"description":  llx.StringData(t.Description),
		"cores":        llx.IntData(int64(t.Cores)),
		"memory":       llx.FloatData(float64(t.Memory)),
		"disk":         llx.IntData(int64(t.Disk)),
		"storageType":  llx.StringData(string(t.StorageType)),
		"cpuType":      llx.StringData(string(t.CPUType)),
		"architecture": llx.StringData(string(t.Architecture)),
		"deprecated":   llx.BoolData(t.IsDeprecated()),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerServerType)
	m.cacheLocations = t.Locations
	return m, nil
}

func (m *mqlHetznerServerType) locations() ([]any, error) {
	out := make([]any, 0, len(m.cacheLocations))
	for _, loc := range m.cacheLocations {
		res, err := newMqlHetznerServerTypeLocation(m.MqlRuntime, m.Id.Data, loc)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// --- serverType.location sub-resource ---

func (r *mqlHetznerServerTypeLocation) id() (string, error) {
	return fmt.Sprintf("hetzner.serverType.location/%d/%d", r.ServerTypeId.Data, r.LocationId.Data), nil
}

func newMqlHetznerServerTypeLocation(runtime *plugin.Runtime, serverTypeID int64, stl hcloud.ServerTypeLocation) (*mqlHetznerServerTypeLocation, error) {
	var locationID int64
	if stl.Location != nil {
		locationID = stl.Location.ID
	}
	dep := map[string]any{}
	if stl.Deprecation != nil {
		dep["announced"] = stl.Deprecation.Announced
		dep["unavailableAfter"] = stl.Deprecation.UnavailableAfter
	}
	res, err := CreateResource(runtime, "hetzner.serverType.location", map[string]*llx.RawData{
		"__id":         llx.StringData(fmt.Sprintf("hetzner.serverType.location/%d/%d", serverTypeID, locationID)),
		"serverTypeId": llx.IntData(serverTypeID),
		"locationId":   llx.IntData(locationID),
		"available":    llx.BoolData(stl.Available),
		"recommended":  llx.BoolData(stl.Recommended),
		"deprecation":  llx.DictData(dep),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerServerTypeLocation), nil
}

func (m *mqlHetznerServerTypeLocation) location() (*mqlHetznerLocation, error) {
	if m.LocationId.Data == 0 {
		m.Location.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ref, err := NewResource(m.MqlRuntime, "hetzner.location", map[string]*llx.RawData{
		"id": llx.IntData(m.LocationId.Data),
	})
	if err != nil {
		return nil, err
	}
	return ref.(*mqlHetznerLocation), nil
}

func initHetznerServerType(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	t, _, err := conn(runtime).Client().ServerType.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, notFoundErr("serverType", id)
	}
	res, err := newMqlHetznerServerType(runtime, t)
	return args, res, err
}

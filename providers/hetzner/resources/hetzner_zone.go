// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlHetznerZone) id() (string, error) {
	return fmt.Sprintf("hetzner.zone/%d", r.Id.Data), nil
}

func (h *mqlHetzner) zones() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Zone, *hcloud.Response, error) {
		return c.Client().Zone.List(ctx(), hcloud.ZoneListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, z := range items {
		res, err := newMqlHetznerZone(h.MqlRuntime, z)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerZone(runtime *plugin.Runtime, z *hcloud.Zone) (*mqlHetznerZone, error) {
	primaryNS := make([]any, 0, len(z.PrimaryNameservers))
	for _, ns := range z.PrimaryNameservers {
		// Intentionally omit ns.TSIGKey — it is a shared secret used for
		// zone transfers and must not be surfaced as queryable data.
		primaryNS = append(primaryNS, map[string]any{
			"address":       ns.Address,
			"port":          int64(ns.Port),
			"tsigAlgorithm": string(ns.TSIGAlgorithm),
		})
	}

	res, err := CreateResource(runtime, "hetzner.zone", map[string]*llx.RawData{
		"__id":                 llx.StringData(fmt.Sprintf("hetzner.zone/%d", z.ID)),
		"id":                   llx.IntData(z.ID),
		"name":                 llx.StringData(z.Name),
		"mode":                 llx.StringData(string(z.Mode)),
		"status":               llx.StringData(string(z.Status)),
		"ttl":                  llx.IntData(int64(z.TTL)),
		"recordCount":          llx.IntData(int64(z.RecordCount)),
		"registrar":            llx.StringData(string(z.Registrar)),
		"primaryNameservers":   dictArrayData(primaryNS),
		"assignedNameservers":  stringArrayData(z.AuthoritativeNameservers.Assigned),
		"delegatedNameservers": stringArrayData(z.AuthoritativeNameservers.Delegated),
		"delegationStatus":     llx.StringData(string(z.AuthoritativeNameservers.DelegationStatus)),
		"delegationLastCheck":  llx.TimeDataPtr(timePtr(z.AuthoritativeNameservers.DelegationLastCheck)),
		"protection":           llx.DictData(protectionDict(z.Protection.Delete)),
		"labels":               labelData(z.Labels),
		"created":              llx.TimeDataPtr(timePtr(z.Created)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerZone), nil
}

func initHetznerZone(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	z, _, err := conn(runtime).Client().Zone.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if z == nil {
		return nil, nil, notFoundErr("zone", id)
	}
	res, err := newMqlHetznerZone(runtime, z)
	return args, res, err
}

func (m *mqlHetznerZone) rrsets() ([]any, error) {
	c := conn(m.MqlRuntime)
	zone := &hcloud.Zone{ID: m.Id.Data}
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.ZoneRRSet, *hcloud.Response, error) {
		return c.Client().Zone.ListRRSets(ctx(), zone, hcloud.ZoneRRSetListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, rr := range items {
		res, err := newMqlHetznerZoneRrset(m.MqlRuntime, m.Id.Data, rr)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlHetznerZoneRrset) id() (string, error) {
	return fmt.Sprintf("hetzner.zone.rrset/%d/%s", r.ZoneId.Data, r.Id.Data), nil
}

func newMqlHetznerZoneRrset(runtime *plugin.Runtime, zoneID int64, rr *hcloud.ZoneRRSet) (*mqlHetznerZoneRrset, error) {
	records := make([]any, 0, len(rr.Records))
	for _, rec := range rr.Records {
		records = append(records, map[string]any{
			"value":   rec.Value,
			"comment": rec.Comment,
		})
	}
	res, err := CreateResource(runtime, "hetzner.zone.rrset", map[string]*llx.RawData{
		"__id":       llx.StringData(fmt.Sprintf("hetzner.zone.rrset/%d/%s", zoneID, rr.ID)),
		"id":         llx.StringData(rr.ID),
		"zoneId":     llx.IntData(zoneID),
		"name":       llx.StringData(rr.Name),
		"type":       llx.StringData(string(rr.Type)),
		"ttl":        llx.IntDataDefault(rr.TTL, 0),
		"records":    dictArrayData(records),
		"protection": llx.DictData(map[string]any{"change": rr.Protection.Change}),
		"labels":     labelData(rr.Labels),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerZoneRrset), nil
}

func (m *mqlHetznerZoneRrset) zone() (*mqlHetznerZone, error) {
	ref, err := NewResource(m.MqlRuntime, "hetzner.zone", map[string]*llx.RawData{
		"id": llx.IntData(m.ZoneId.Data),
	})
	if err != nil {
		return nil, err
	}
	return ref.(*mqlHetznerZone), nil
}

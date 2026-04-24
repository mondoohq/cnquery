// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerNetworkInternal struct {
	cacheServers []*hcloud.Server
}

func (r *mqlHetznerNetwork) id() (string, error) {
	return fmt.Sprintf("hetzner.network/%d", r.Id.Data), nil
}

func (h *mqlHetzner) networks() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Network, *hcloud.Response, error) {
		return c.Client().Network.List(ctx(), hcloud.NetworkListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, n := range items {
		res, err := newMqlHetznerNetwork(h.MqlRuntime, n)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerNetwork(runtime *plugin.Runtime, n *hcloud.Network) (*mqlHetznerNetwork, error) {
	subnets := make([]any, 0, len(n.Subnets))
	for _, s := range n.Subnets {
		subnets = append(subnets, map[string]any{
			"type":        string(s.Type),
			"ipRange":     ipNetString(s.IPRange),
			"networkZone": string(s.NetworkZone),
			"gateway":     ipString(s.Gateway),
		})
	}
	routes := make([]any, 0, len(n.Routes))
	for _, r := range n.Routes {
		routes = append(routes, map[string]any{
			"destination": ipNetString(r.Destination),
			"gateway":     ipString(r.Gateway),
		})
	}

	res, err := CreateResource(runtime, "hetzner.network", map[string]*llx.RawData{
		"__id":                  llx.StringData(fmt.Sprintf("hetzner.network/%d", n.ID)),
		"id":                    llx.IntData(n.ID),
		"name":                  llx.StringData(n.Name),
		"ipRange":               llx.StringData(ipNetString(n.IPRange)),
		"created":               llx.TimeDataPtr(timePtr(n.Created)),
		"subnets":               dictArrayData(subnets),
		"routes":                dictArrayData(routes),
		"exposeRoutesToVswitch": llx.BoolData(n.ExposeRoutesToVSwitch),
		"protection":            llx.DictData(protectionDict(n.Protection.Delete)),
		"labels":                labelData(n.Labels),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerNetwork)
	m.cacheServers = n.Servers
	return m, nil
}

func initHetznerNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	n, _, err := conn(runtime).Client().Network.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if n == nil {
		return nil, nil, notFoundErr("network", id)
	}
	res, err := newMqlHetznerNetwork(runtime, n)
	return args, res, err
}

func (m *mqlHetznerNetwork) servers() ([]any, error) {
	out := make([]any, 0, len(m.cacheServers))
	for _, s := range m.cacheServers {
		// Network.Servers is a list of partial Server objects (just IDs).
		// Use NewResource so an init-by-id can fully resolve them on demand.
		ref, err := NewResource(m.MqlRuntime, "hetzner.server", map[string]*llx.RawData{
			"id": llx.IntData(s.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

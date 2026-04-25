// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerFirewallInternal struct {
	cacheServerIDs []int64
}

func (r *mqlHetznerFirewall) id() (string, error) {
	return fmt.Sprintf("hetzner.firewall/%d", r.Id.Data), nil
}

func (h *mqlHetzner) firewalls() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Firewall, *hcloud.Response, error) {
		return c.Client().Firewall.List(ctx(), hcloud.FirewallListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, fw := range items {
		res, err := newMqlHetznerFirewall(h.MqlRuntime, fw)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerFirewall(runtime *plugin.Runtime, fw *hcloud.Firewall) (*mqlHetznerFirewall, error) {
	rules := make([]any, 0, len(fw.Rules))
	for _, r := range fw.Rules {
		srcs := make([]any, 0, len(r.SourceIPs))
		for _, ip := range r.SourceIPs {
			srcs = append(srcs, ip.String())
		}
		dsts := make([]any, 0, len(r.DestinationIPs))
		for _, ip := range r.DestinationIPs {
			dsts = append(dsts, ip.String())
		}
		entry := map[string]any{
			"direction":      string(r.Direction),
			"protocol":       string(r.Protocol),
			"sourceIps":      srcs,
			"destinationIps": dsts,
		}
		if r.Port != nil {
			entry["port"] = *r.Port
		}
		if r.Description != nil {
			entry["description"] = *r.Description
		}
		rules = append(rules, entry)
	}

	var serverIDs []int64
	var labelSelectors []string
	for _, a := range fw.AppliedTo {
		if a.Server != nil {
			serverIDs = append(serverIDs, a.Server.ID)
		}
		if a.LabelSelector != nil {
			labelSelectors = append(labelSelectors, a.LabelSelector.Selector)
		}
	}

	res, err := CreateResource(runtime, "hetzner.firewall", map[string]*llx.RawData{
		"__id":           llx.StringData(fmt.Sprintf("hetzner.firewall/%d", fw.ID)),
		"id":             llx.IntData(fw.ID),
		"name":           llx.StringData(fw.Name),
		"created":        llx.TimeDataPtr(timePtr(fw.Created)),
		"rules":          dictArrayData(rules),
		"labelSelectors": stringArrayData(labelSelectors),
		"labels":         labelData(fw.Labels),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerFirewall)
	m.cacheServerIDs = serverIDs
	return m, nil
}

func initHetznerFirewall(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	fw, _, err := conn(runtime).Client().Firewall.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if fw == nil {
		return nil, nil, notFoundErr("firewall", id)
	}
	res, err := newMqlHetznerFirewall(runtime, fw)
	return args, res, err
}

func (m *mqlHetznerFirewall) servers() ([]any, error) {
	out := make([]any, 0, len(m.cacheServerIDs))
	for _, id := range m.cacheServerIDs {
		ref, err := NewResource(m.MqlRuntime, "hetzner.server", map[string]*llx.RawData{
			"id": llx.IntData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

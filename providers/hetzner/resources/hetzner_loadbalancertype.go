// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlHetznerLoadBalancerType) id() (string, error) {
	return fmt.Sprintf("hetzner.loadBalancerType/%d", r.Id.Data), nil
}

func (h *mqlHetzner) loadBalancerTypes() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.LoadBalancerType, *hcloud.Response, error) {
		return c.Client().LoadBalancerType.List(ctx(), hcloud.LoadBalancerTypeListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, t := range items {
		res, err := newMqlHetznerLoadBalancerType(h.MqlRuntime, t)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerLoadBalancerType(runtime *plugin.Runtime, t *hcloud.LoadBalancerType) (*mqlHetznerLoadBalancerType, error) {
	// The legacy Deprecated string is populated by the SDK from the announcement
	// date, so `deprecated` is read straight from the structured Deprecation
	// info. That preserves the field's meaning while dropping the round-trip
	// through an RFC3339 string that the API no longer returns.
	var announced *time.Time
	if t.Deprecation != nil {
		announced = timePtr(t.Deprecation.Announced)
	}
	res, err := CreateResource(runtime, "hetzner.loadBalancerType", map[string]*llx.RawData{
		"__id":                    llx.StringData(fmt.Sprintf("hetzner.loadBalancerType/%d", t.ID)),
		"id":                      llx.IntData(t.ID),
		"name":                    llx.StringData(t.Name),
		"description":             llx.StringData(t.Description),
		"maxConnections":          llx.IntData(int64(t.MaxConnections)),
		"maxServices":             llx.IntData(int64(t.MaxServices)),
		"maxTargets":              llx.IntData(int64(t.MaxTargets)),
		"maxAssignedCertificates": llx.IntData(int64(t.MaxAssignedCertificates)),
		"deprecated":              llx.TimeDataPtr(announced),
		"deprecation":             llx.DictData(deprecationDict(t.Deprecation)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerLoadBalancerType), nil
}

func initHetznerLoadBalancerType(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	t, _, err := conn(runtime).Client().LoadBalancerType.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, notFoundErr("loadBalancerType", id)
	}
	res, err := newMqlHetznerLoadBalancerType(runtime, t)
	return args, res, err
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlStackit) loadBalancers() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.LoadBalancer()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListLoadBalancersExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetLoadBalancersOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildLoadBalancer(r.MqlRuntime, &items[i], c.Region())
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// mqlStackitLoadBalancerInternal caches the raw listener and target-pool
// slices so the computed listeners()/targetPools() accessors can build
// typed sub-resources on access without re-fetching the load balancer.
type mqlStackitLoadBalancerInternal struct {
	rawListeners   []loadbalancer.Listener
	rawTargetPools []loadbalancer.TargetPool
}

func buildLoadBalancer(runtime *plugin.Runtime, lb *loadbalancer.LoadBalancer, region string) (plugin.Resource, error) {
	status := string(lb.GetStatus())
	var privateOnly bool
	if v, ok := lb.GetOptionsOk(); ok {
		if pno, ok := v.GetPrivateNetworkOnlyOk(); ok {
			privateOnly = pno
		}
	}

	args := map[string]*llx.RawData{
		"name":               llx.StringData(lb.GetName()),
		"externalAddress":    llx.StringData(lb.GetExternalAddress()),
		"planId":             llx.StringData(lb.GetPlanId()),
		"status":             llx.StringData(status),
		"privateNetworkOnly": llx.BoolData(privateOnly),
		"region":             llx.StringData(region),
		"networks":           llx.ArrayData(anySliceToDict(lb.GetNetworks()), types.Dict),
		"options":            llx.DictData(toDict(lb.GetOptions())),
		"errors":             llx.ArrayData(anySliceToDict(lb.GetErrors()), types.Dict),
	}
	res, err := CreateResource(runtime, "stackit.loadBalancer", args)
	if err != nil {
		return nil, err
	}
	if mlb, ok := res.(*mqlStackitLoadBalancer); ok {
		mlb.rawListeners = lb.GetListeners()
		mlb.rawTargetPools = lb.GetTargetPools()
	}
	return res, nil
}

func (r *mqlStackitLoadBalancer) listeners() ([]any, error) {
	out := make([]any, 0, len(r.rawListeners))
	for i := range r.rawListeners {
		l := &r.rawListeners[i]
		args := map[string]*llx.RawData{
			"name":                 llx.StringData(l.GetName()),
			"displayName":          llx.StringData(l.GetDisplayName()),
			"loadBalancerName":     llx.StringData(r.Name.Data),
			"port":                 llx.IntData(l.GetPort()),
			"protocol":             llx.StringData(string(l.GetProtocol())),
			"targetPool":           llx.StringData(l.GetTargetPool()),
			"serverNameIndicators": strSliceData(listenerSNINames(l.GetServerNameIndicators())),
			"tcp":                  llx.DictData(toDict(l.GetTcp())),
			"udp":                  llx.DictData(toDict(l.GetUdp())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.loadBalancer.listener", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// listenerSNINames extracts the Name field from each ServerNameIndicator
// struct so the schema can expose them as []string.
func listenerSNINames(snis []loadbalancer.ServerNameIndicator) []string {
	out := make([]string, 0, len(snis))
	for i := range snis {
		out = append(out, snis[i].GetName())
	}
	return out
}

func (r *mqlStackitLoadBalancer) targetPools() ([]any, error) {
	out := make([]any, 0, len(r.rawTargetPools))
	for i := range r.rawTargetPools {
		tp := &r.rawTargetPools[i]
		args := map[string]*llx.RawData{
			"name":               llx.StringData(tp.GetName()),
			"loadBalancerName":   llx.StringData(r.Name.Data),
			"targetPort":         llx.IntData(tp.GetTargetPort()),
			"targets":            llx.ArrayData(anySliceToDict(tp.GetTargets()), types.Dict),
			"activeHealthCheck":  llx.DictData(toDict(tp.GetActiveHealthCheck())),
			"sessionPersistence": llx.DictData(toDict(tp.GetSessionPersistence())),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.loadBalancer.targetPool", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitLoadBalancerListener) id() (string, error) {
	return "stackit.loadBalancer.listener/" + r.LoadBalancerName.Data + "/" + r.Name.Data, nil
}

func (r *mqlStackitLoadBalancerTargetPool) id() (string, error) {
	return "stackit.loadBalancer.targetPool/" + r.LoadBalancerName.Data + "/" + r.Name.Data, nil
}

func (r *mqlStackitLoadBalancer) id() (string, error) {
	return "stackit.loadBalancer/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

func initStackitLoadBalancer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	v, ok := args["name"]
	if !ok || v == nil {
		return args, nil, nil
	}
	name, ok := v.Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.LoadBalancer()
	if err != nil {
		return nil, nil, err
	}
	lb, err := client.GetLoadBalancerExecute(bgctx(), c.ProjectID(), c.Region(), name)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildLoadBalancer(runtime, lb, c.Region())
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

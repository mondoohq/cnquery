// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/alb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlStackit) albLoadBalancers() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ALB()
	if err != nil {
		return nil, err
	}
	out := []any{}
	pageId := ""
	for {
		req := client.ListLoadBalancers(bgctx(), c.ProjectID(), c.Region())
		if pageId != "" {
			req = req.PageId(pageId)
		}
		resp, err := req.Execute()
		if err != nil {
			if isAccessDenied(err) {
				return []any{}, nil
			}
			return nil, err
		}
		items, _ := resp.GetLoadBalancersOk()
		for i := range items {
			res, err := buildAlbLoadBalancer(r.MqlRuntime, &items[i], c.Region())
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		next, ok := resp.GetNextPageIdOk()
		if !ok || next == "" {
			break
		}
		pageId = next
	}
	return out, nil
}

func buildAlbLoadBalancer(runtime *plugin.Runtime, lb *alb.LoadBalancer, region string) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"name":                                 llx.StringData(lb.GetName()),
		"externalAddress":                      llx.StringData(lb.GetExternalAddress()),
		"privateAddress":                       llx.StringData(lb.GetPrivateAddress()),
		"planId":                               llx.StringData(lb.GetPlanId()),
		"status":                               llx.StringData(string(lb.GetStatus())),
		"region":                               llx.StringData(region),
		"listeners":                            llx.ArrayData(anySliceToDict(lb.GetListeners()), types.Dict),
		"networks":                             llx.ArrayData(anySliceToDict(lb.GetNetworks()), types.Dict),
		"targetPools":                          llx.ArrayData(anySliceToDict(lb.GetTargetPools()), types.Dict),
		"options":                              llx.DictData(toDict(lb.GetOptions())),
		"errors":                               llx.ArrayData(anySliceToDict(lb.GetErrors()), types.Dict),
		"loadBalancerSecurityGroup":            llx.DictData(toDict(lb.GetLoadBalancerSecurityGroup())),
		"targetSecurityGroup":                  llx.DictData(toDict(lb.GetTargetSecurityGroup())),
		"disableTargetSecurityGroupAssignment": llx.BoolData(lb.GetDisableTargetSecurityGroupAssignment()),
		"labels":                               labelData(lb.GetLabels()),
	}
	return CreateResource(runtime, "stackit.alb.loadBalancer", args)
}

func (r *mqlStackitAlbLoadBalancer) id() (string, error) {
	return "stackit.alb.loadBalancer/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

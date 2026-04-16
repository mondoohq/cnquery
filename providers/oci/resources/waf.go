// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/waf"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciWaf) id() (string, error) {
	return "oci.waf", nil
}

func (o *mqlOciWaf) firewalls() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getFirewalls(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciWaf) getFirewalls(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci WAF firewalls with region %s", regionResource.Id.Data)

			svc, err := conn.WafClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []waf.WebAppFirewallSummary
			var page *string
			for {
				response, err := svc.ListWebAppFirewalls(ctx, waf.ListWebAppFirewallsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				items = append(items, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for _, item := range items {
				// The summary is an interface; we handle the LoadBalancer concrete type
				lbWaf, ok := item.(waf.WebAppFirewallLoadBalancerSummary)
				if !ok {
					continue
				}

				var created *time.Time
				if lbWaf.TimeCreated != nil {
					created = &lbWaf.TimeCreated.Time
				}
				var timeUpdated *time.Time
				if lbWaf.TimeUpdated != nil {
					timeUpdated = &lbWaf.TimeUpdated.Time
				}

				freeformTags := make(map[string]interface{}, len(lbWaf.FreeformTags))
				for k, v := range lbWaf.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(lbWaf.DefinedTags))
				for k, v := range lbWaf.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.waf.firewall", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(lbWaf.Id),
					"name":          llx.StringDataPtr(lbWaf.DisplayName),
					"compartmentID": llx.StringDataPtr(lbWaf.CompartmentId),
					"state":         llx.StringData(string(lbWaf.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"timeUpdated":   llx.TimeDataPtr(timeUpdated),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlFw := mqlInstance.(*mqlOciWafFirewall)
				mqlFw.cachePolicyId = stringValue(lbWaf.WebAppFirewallPolicyId)
				mqlFw.cacheLoadBalancerId = stringValue(lbWaf.LoadBalancerId)
				res = append(res, mqlFw)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciWafFirewallInternal struct {
	cachePolicyId       string
	cacheLoadBalancerId string
}

func (o *mqlOciWafFirewall) id() (string, error) {
	return "oci.waf.firewall/" + o.Id.Data, nil
}

func (o *mqlOciWafFirewall) policy() (*mqlOciWafPolicy, error) {
	if o.cachePolicyId == "" {
		o.Policy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlPolicy, err := NewResource(o.MqlRuntime, "oci.waf.policy", map[string]*llx.RawData{
		"id": llx.StringData(o.cachePolicyId),
	})
	if err != nil {
		return nil, err
	}
	return mqlPolicy.(*mqlOciWafPolicy), nil
}

func (o *mqlOciWafFirewall) loadBalancer() (*mqlOciLoadBalancerLoadBalancer, error) {
	if o.cacheLoadBalancerId == "" {
		o.LoadBalancer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlLb, err := NewResource(o.MqlRuntime, "oci.loadBalancer.loadBalancer", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheLoadBalancerId),
	})
	if err != nil {
		return nil, err
	}
	return mqlLb.(*mqlOciLoadBalancerLoadBalancer), nil
}

func (o *mqlOciWaf) policies() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getPolicies(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciWaf) getPolicies(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci WAF policies with region %s", regionResource.Id.Data)

			svc, err := conn.WafClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []waf.WebAppFirewallPolicySummary
			var page *string
			for {
				response, err := svc.ListWebAppFirewallPolicies(ctx, waf.ListWebAppFirewallPoliciesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				items = append(items, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range items {
				p := items[i]

				var created *time.Time
				if p.TimeCreated != nil {
					created = &p.TimeCreated.Time
				}
				var timeUpdated *time.Time
				if p.TimeUpdated != nil {
					timeUpdated = &p.TimeUpdated.Time
				}

				freeformTags := make(map[string]interface{}, len(p.FreeformTags))
				for k, v := range p.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(p.DefinedTags))
				for k, v := range p.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.waf.policy", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(p.Id),
					"name":          llx.StringDataPtr(p.DisplayName),
					"compartmentID": llx.StringDataPtr(p.CompartmentId),
					"state":         llx.StringData(string(p.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"timeUpdated":   llx.TimeDataPtr(timeUpdated),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciWafPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.waf.policy")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.waf", nil)
	if err != nil {
		return nil, nil, err
	}
	w := obj.(*mqlOciWaf)

	rawPolicies := w.GetPolicies()
	if rawPolicies.Error != nil {
		return nil, nil, rawPolicies.Error
	}

	for _, raw := range rawPolicies.Data {
		p := raw.(*mqlOciWafPolicy)
		if p.Id.Data == idVal {
			return args, p, nil
		}
	}

	return nil, nil, errors.New("oci.waf.policy not found: " + idVal)
}

func (o *mqlOciWafPolicy) id() (string, error) {
	return "oci.waf.policy/" + o.Id.Data, nil
}

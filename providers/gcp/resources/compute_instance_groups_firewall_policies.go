// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Instance groups

func (g *mqlGcpProjectComputeService) instanceGroups() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.InstanceGroups.AggregatedList(projectId)
	if err := req.Pages(ctx, func(page *compute.InstanceGroupAggregatedList) error {
		for _, scoped := range page.Items {
			for _, ig := range scoped.InstanceGroups {
				namedPorts, _ := convert.JsonToDictSlice(ig.NamedPorts)

				mqlIG, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.instanceGroup", map[string]*llx.RawData{
					"id":          llx.StringData(strconv.FormatUint(ig.Id, 10)),
					"projectId":   llx.StringData(projectId),
					"name":        llx.StringData(ig.Name),
					"description": llx.StringData(ig.Description),
					"zoneUrl":     llx.StringData(ig.Zone),
					"networkUrl":  llx.StringData(ig.Network),
					"size":        llx.IntData(ig.Size),
					"namedPorts":  llx.ArrayData(namedPorts, types.Dict),
					"created":     llx.TimeDataPtr(parseTime(ig.CreationTimestamp)),
					"selfLink":    llx.StringData(ig.SelfLink),
				})
				if err != nil {
					return err
				}
				mqlIG.(*mqlGcpProjectComputeServiceInstanceGroup).cacheSubnetworkUrl = ig.Subnetwork
				res = append(res, mqlIG)
			}
		}
		return nil
	}); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 403 {
			log.Warn().Str("project", projectId).Err(err).Msg("could not list compute instance groups")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

type mqlGcpProjectComputeServiceInstanceGroupInternal struct {
	cacheSubnetworkUrl string
}

func (g *mqlGcpProjectComputeServiceInstanceGroup) id() (string, error) {
	return "gcloud.compute.instanceGroup/" + g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeServiceInstanceGroup) network() (*mqlGcpProjectComputeServiceNetwork, error) {
	return getNetworkByUrl(g.NetworkUrl.Data, g.MqlRuntime)
}

func (g *mqlGcpProjectComputeServiceInstanceGroup) subnetwork() (*mqlGcpProjectComputeServiceSubnetwork, error) {
	return getSubnetworkByUrl(g.cacheSubnetworkUrl, g.MqlRuntime)
}

// Instance group managers

func (g *mqlGcpProjectComputeService) instanceGroupManagers() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.InstanceGroupManagers.AggregatedList(projectId)
	if err := req.Pages(ctx, func(page *compute.InstanceGroupManagerAggregatedList) error {
		for _, scoped := range page.Items {
			for _, igm := range scoped.InstanceGroupManagers {
				currentActions, _ := convert.JsonToDict(igm.CurrentActions)
				statefulPolicy, _ := convert.JsonToDict(igm.StatefulPolicy)
				autoHealingPolicies, _ := convert.JsonToDictSlice(igm.AutoHealingPolicies)
				status, _ := convert.JsonToDict(igm.Status)

				mqlIGM, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.instanceGroupManager", map[string]*llx.RawData{
					"id":                  llx.StringData(strconv.FormatUint(igm.Id, 10)),
					"projectId":           llx.StringData(projectId),
					"name":                llx.StringData(igm.Name),
					"description":         llx.StringData(igm.Description),
					"zoneUrl":             llx.StringData(igm.Zone),
					"regionUrl":           llx.StringData(igm.Region),
					"instanceTemplateUrl": llx.StringData(igm.InstanceTemplate),
					"targetSize":          llx.IntData(igm.TargetSize),
					"currentActions":      llx.DictData(currentActions),
					"statefulPolicy":      llx.DictData(statefulPolicy),
					"autoHealingPolicies": llx.ArrayData(autoHealingPolicies, types.Dict),
					"instanceGroupUrl":    llx.StringData(igm.InstanceGroup),
					"status":              llx.DictData(status),
					"created":             llx.TimeDataPtr(parseTime(igm.CreationTimestamp)),
					"selfLink":            llx.StringData(igm.SelfLink),
					"baseInstanceName":    llx.StringData(igm.BaseInstanceName),
				})
				if err != nil {
					return err
				}
				res = append(res, mqlIGM)
			}
		}
		return nil
	}); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 403 {
			log.Warn().Str("project", projectId).Err(err).Msg("could not list compute instance group managers")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceInstanceGroupManager) id() (string, error) {
	return "gcloud.compute.instanceGroupManager/" + g.Id.Data, g.Id.Error
}

// Network firewall policies

func (g *mqlGcpProjectComputeService) firewallPolicies() ([]any, error) {
	if !g.GetEnabled().Data {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.NetworkFirewallPolicies.List(projectId)
	if err := req.Pages(ctx, func(page *compute.FirewallPolicyList) error {
		for _, fp := range page.Items {
			associations, _ := convert.JsonToDictSlice(fp.Associations)

			mqlFP, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.firewallPolicy", map[string]*llx.RawData{
				"id":             llx.StringData(strconv.FormatUint(fp.Id, 10)),
				"projectId":      llx.StringData(projectId),
				"name":           llx.StringData(fp.ShortName),
				"displayName":    llx.StringData(fp.DisplayName),
				"description":    llx.StringData(fp.Description),
				"selfLink":       llx.StringData(fp.SelfLink),
				"ruleTupleCount": llx.IntData(fp.RuleTupleCount),
				"created":        llx.TimeDataPtr(parseTime(fp.CreationTimestamp)),
				"regionUrl":      llx.StringData(fp.Region),
				"associations":   llx.ArrayData(associations, types.Dict),
			})
			if err != nil {
				return err
			}
			mqlPolicy := mqlFP.(*mqlGcpProjectComputeServiceFirewallPolicy)
			mqlPolicy.cacheRules = fp.Rules
			res = append(res, mqlFP)
		}
		return nil
	}); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 403 {
			log.Warn().Str("project", projectId).Err(err).Msg("could not list compute network firewall policies")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

type mqlGcpProjectComputeServiceFirewallPolicyInternal struct {
	cacheRules []*compute.FirewallPolicyRule
}

func (g *mqlGcpProjectComputeServiceFirewallPolicy) id() (string, error) {
	return "gcloud.compute.firewallPolicy/" + g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeServiceFirewallPolicy) fetchRules() ([]*compute.FirewallPolicyRule, error) {
	projectId := g.ProjectId.Data
	name := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	policy, err := computeSvc.NetworkFirewallPolicies.Get(projectId, name).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return policy.Rules, nil
}

func (g *mqlGcpProjectComputeServiceFirewallPolicy) rules() ([]any, error) {
	if g.cacheRules == nil {
		// If the resource was resolved from cache rather than through the list,
		// fetch the policy from the API to get its rules.
		rules, err := g.fetchRules()
		if err != nil {
			return nil, err
		}
		g.cacheRules = rules
		if g.cacheRules == nil {
			return nil, nil
		}
	}
	policyId := g.Id.Data
	res := make([]any, 0, len(g.cacheRules))
	for _, r := range g.cacheRules {
		match, _ := convert.JsonToDict(r.Match)

		mqlRule, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.firewallPolicy.rule", map[string]*llx.RawData{
			"id":                    llx.StringData(fmt.Sprintf("%s/rule/%d", policyId, r.Priority)),
			"priority":              llx.IntData(int64(r.Priority)),
			"action":                llx.StringData(r.Action),
			"direction":             llx.StringData(r.Direction),
			"description":           llx.StringData(r.Description),
			"disabled":              llx.BoolData(r.Disabled),
			"enableLogging":         llx.BoolData(r.EnableLogging),
			"match":                 llx.DictData(match),
			"targetResources":       llx.ArrayData(convert.SliceAnyToInterface(r.TargetResources), types.String),
			"targetServiceAccounts": llx.ArrayData(convert.SliceAnyToInterface(r.TargetServiceAccounts), types.String),
			"ruleName":              llx.StringData(r.RuleName),
			"securityProfileGroup":  llx.StringData(r.SecurityProfileGroup),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceFirewallPolicyRule) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

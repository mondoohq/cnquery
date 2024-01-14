// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectDnsService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	id := g.ProjectId.Data
	return "gcp.project.dnsService/" + id, nil
}

func (g *mqlGcpProject) dns() (*mqlGcpProjectDnsService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectDnsService), nil
}

func initGcpProjectDnsService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcp.project.dnsService.managedzone/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsService) managedZones() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(dns.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	dnsSvc, err := dns.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	req := dnsSvc.ManagedZones.List(projectId)
	if err := req.Pages(ctx, func(page *dns.ManagedZonesListResponse) error {
		for i := range page.ManagedZones {
			managedZone := page.ManagedZones[i]

			var mqlDnssecCfg map[string]interface{}
			if managedZone.DnssecConfig != nil {
				keySpecs := make([]interface{}, 0, len(managedZone.DnssecConfig.DefaultKeySpecs))
				for _, keySpec := range managedZone.DnssecConfig.DefaultKeySpecs {
					keySpecs = append(keySpecs, map[string]interface{}{
						"algorithm": keySpec.Algorithm,
						"keyLength": keySpec.KeyLength,
						"keyType":   keySpec.KeyType,
					})
				}
				mqlDnssecCfg = map[string]interface{}{
					"defaultKeySpecs": keySpecs,
					"nonExistence":    managedZone.DnssecConfig.NonExistence,
					"state":           managedZone.DnssecConfig.State,
				}
			}

			mqlManagedZone, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService.managedzone", map[string]*llx.RawData{
				"id":            llx.StringData(strconv.FormatInt(int64(managedZone.Id), 10)),
				"projectId":     llx.StringData(projectId),
				"name":          llx.StringData(managedZone.Name),
				"description":   llx.StringData(managedZone.Description),
				"dnssecConfig":  llx.DictData(mqlDnssecCfg),
				"dnsName":       llx.StringData(managedZone.DnsName),
				"nameServerSet": llx.StringData(managedZone.NameServerSet),
				"nameServers":   llx.ArrayData(convert.SliceAnyToInterface(managedZone.NameServers), types.String),
				"visibility":    llx.StringData(managedZone.Visibility),
				"created":       llx.TimeDataPtr(parseTime(managedZone.CreationTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlManagedZone)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectDnsServicePolicy) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcp.project.dnsService.policy/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsService) policies() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(dns.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	dnsSvc, err := dns.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	req := dnsSvc.Policies.List(projectId)
	if err := req.Pages(ctx, func(page *dns.PoliciesListResponse) error {
		for i := range page.Policies {
			policy := page.Policies[i]

			networkNames := make([]interface{}, 0, len(policy.Networks))
			for _, network := range policy.Networks {
				segments := strings.Split(network.NetworkUrl, "/")
				networkNames = append(networkNames, segments[len(segments)-1])
			}

			mqlDnsPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService.policy", map[string]*llx.RawData{
				"projectId":               llx.StringData(projectId),
				"id":                      llx.StringData(strconv.FormatInt(int64(policy.Id), 10)),
				"name":                    llx.StringData(policy.Name),
				"description":             llx.StringData(policy.Description),
				"enableInboundForwarding": llx.BoolData(policy.EnableInboundForwarding),
				"enableLogging":           llx.BoolData(policy.EnableLogging),
				"networkNames":            llx.ArrayData(networkNames, types.String),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlDnsPolicy)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectDnsServicePolicy) networks() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	networkNames := g.GetNetworkNames()
	if networkNames.Error != nil {
		return nil, networkNames.Error
	}

	obj, err := CreateResource(g.MqlRuntime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	gcpCompute := obj.(*mqlGcpProjectComputeService)
	networks := gcpCompute.GetNetworks()
	if networks.Error != nil {
		return nil, networks.Error
	}

	res := make([]interface{}, 0, len(networkNames.Data))
	for _, network := range networks.Data {
		networkName := network.(*mqlGcpProjectComputeServiceNetwork).Name.Data
		for _, name := range networkNames.Data {
			if name == networkName {
				res = append(res, network)
				break
			}
		}
	}
	return res, nil
}

func (g *mqlGcpProjectDnsServiceRecordset) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	id := g.Name.Data
	return "gcp.project.dnsService.recordset/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) recordSets() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	managedZone := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(dns.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	dnsSvc, err := dns.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	req := dnsSvc.ResourceRecordSets.List(projectId, managedZone)
	if err := req.Pages(ctx, func(page *dns.ResourceRecordSetsListResponse) error {
		for i := range page.Rrsets {
			rSet := page.Rrsets[i]

			mqlDnsPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService.recordset", map[string]*llx.RawData{
				"projectId":        llx.StringData(projectId),
				"name":             llx.StringData(rSet.Name),
				"rrdatas":          llx.ArrayData(convert.SliceAnyToInterface(rSet.Rrdatas), types.String),
				"signatureRrdatas": llx.ArrayData(convert.SliceAnyToInterface(rSet.SignatureRrdatas), types.String),
				"ttl":              llx.IntData(rSet.Ttl),
				"type":             llx.StringData(rSet.Type),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlDnsPolicy)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

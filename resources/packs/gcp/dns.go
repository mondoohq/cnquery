package gcp

import (
	"context"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/resources"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectDnsService) id() (string, error) {
	id, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return "gcp.project.dnsService/" + id, nil
}

func (g *mqlGcpProject) GetDns() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.dnsService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectDnsService) init(args *resources.Args) (*resources.Args, GcpProjectDnsService, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.project.dnsService.managedzone/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsService) GetManagedZones() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dns.CloudPlatformReadOnlyScope)
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

			mqlManagedZone, err := g.MotorRuntime.CreateResource("gcp.project.dnsService.managedzone",
				"id", strconv.FormatInt(int64(managedZone.Id), 10),
				"projectId", projectId,
				"name", managedZone.Name,
				"description", managedZone.Description,
				"dnssecConfig", mqlDnssecCfg,
				"dnsName", managedZone.DnsName,
				"nameServerSet", managedZone.NameServerSet,
				"nameServers", core.SliceToInterfaceSlice(managedZone.NameServers),
				"visibility", managedZone.Visibility,
				"created", parseTime(managedZone.CreationTime),
			)
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
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.project.dnsService.policy/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsService) GetPolicies() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dns.CloudPlatformReadOnlyScope)
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

			mqlDnsPolicy, err := g.MotorRuntime.CreateResource("gcp.project.dnsService.policy",
				"projectId", projectId,
				"id", strconv.FormatInt(int64(policy.Id), 10),
				"name", policy.Name,
				"description", policy.Description,
				"enableInboundForwarding", policy.EnableInboundForwarding,
				"enableLogging", policy.EnableLogging,
				"networkNames", networkNames,
			)
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

func (g *mqlGcpProjectDnsServicePolicy) GetNetworks() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	networkNames, err := g.NetworkNames()
	if err != nil {
		return nil, err
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", projectId)
	if err != nil {
		return nil, err
	}
	gcpCompute := obj.(GcpProjectComputeService)
	networks, err := gcpCompute.Networks()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(networkNames))
	for _, network := range networks {
		networkName, err := network.(GcpProjectComputeServiceNetwork).Name()
		if err != nil {
			return nil, err
		}
		for _, name := range networkNames {
			if name == networkName {
				res = append(res, network)
				break
			}
		}
	}
	return res, nil
}

func (g *mqlGcpProjectDnsServiceRecordset) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	id, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.project.dnsService.recordset/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) GetRecordSets() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	managedZone, err := g.Name()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dns.CloudPlatformReadOnlyScope)
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

			mqlDnsPolicy, err := g.MotorRuntime.CreateResource("gcp.project.dnsService.recordset",
				"projectId", projectId,
				"name", rSet.Name,
				"rrdatas", core.SliceToInterfaceSlice(rSet.Rrdatas),
				"signatureRrdatas", core.SliceToInterfaceSlice(rSet.SignatureRrdatas),
				"ttl", rSet.Ttl,
				"type", rSet.Type,
			)
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

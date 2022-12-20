package gcp

import (
	"context"
	"strconv"

	"go.mondoo.com/cnquery/resources"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpDns) id() (string, error) {
	id, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return "gcp.dns/" + id, nil
}

func (g *mqlGcpDns) init(args *resources.Args) (*resources.Args, GcpDns, error) {
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

func (g *mqlGcpDnsManagedzone) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.dns.managedzone/" + projectId + "/" + id, nil
}

func (g *mqlGcpDns) GetManagedZones() ([]interface{}, error) {
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

			mqlManagedZone, err := g.MotorRuntime.CreateResource("gcp.dns.managedzone",
				"id", strconv.FormatInt(int64(managedZone.Id), 10),
				"projectId", projectId,
				"name", managedZone.Name,
				"description", managedZone.Description,
				"dnsName", managedZone.DnsName,
				"nameServerSet", managedZone.NameServerSet,
				"nameServers", core.StrSliceToInterface(managedZone.NameServers),
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

func (g *mqlGcpDnsPolicy) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.dns.policy/" + projectId + "/" + id, nil
}

func (g *mqlGcpDns) GetPolicies() ([]interface{}, error) {
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

			mqlDnsPolicy, err := g.MotorRuntime.CreateResource("gcp.dns.policy",
				"projectId", projectId,
				"id", strconv.FormatInt(int64(policy.Id), 10),
				"name", policy.Name,
				"description", policy.Description,
				"enableInboundForwarding", policy.EnableInboundForwarding,
				"enableLogging", policy.EnableLogging,
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

func (g *mqlGcpDnsRecordset) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	id, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.dns.recordset/" + projectId + "/" + id, nil
}

func (g *mqlGcpDnsManagedzone) GetRecordSets() ([]interface{}, error) {
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

			mqlDnsPolicy, err := g.MotorRuntime.CreateResource("gcp.dns.recordset",
				"projectId", projectId,
				"name", rSet.Name,
				"rrdatas", core.StrSliceToInterface(rSet.Rrdatas),
				"signatureRrdatas", core.StrSliceToInterface(rSet.SignatureRrdatas),
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

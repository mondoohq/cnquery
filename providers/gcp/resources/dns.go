// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

func initGcpProjectDnsServiceManagedzone(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	// Create the parent DNS service and find the specific managed zone
	obj, err := CreateResource(runtime, "gcp.project.dnsService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	dnsSvc := obj.(*mqlGcpProjectDnsService)
	managedzones := dnsSvc.GetManagedZones()
	if managedzones.Error != nil {
		return nil, nil, managedzones.Error
	}

	// Find the matching managed zone
	for _, mz := range managedzones.Data {
		managedzone := mz.(*mqlGcpProjectDnsServiceManagedzone)
		id := managedzone.GetId()
		if id.Error != nil {
			return nil, nil, id.Error
		}
		projectId := managedzone.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

		if id.Data == args["name"].Value && projectId.Data == args["projectId"].Value {
			return args, managedzone, nil
		}
	}

	return nil, nil, errors.New("DNS managed zone not found")
}

type mqlGcpProjectDnsServiceInternal struct {
	serviceEnabled bool
	serviceChecked bool
}

// managedZoneDnssecNonExistence returns the DNSSEC proof-of-nonexistence mode
// ("nsec" or "nsec3") for a managed zone, or "" when DNSSEC is not configured.
func managedZoneDnssecNonExistence(cfg *dns.ManagedZoneDnsSecConfig) string {
	if cfg == nil {
		return ""
	}
	return cfg.NonExistence
}

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

	serviceEnabled, err := g.isServiceEnabled(service_dns)
	if err != nil {
		return nil, err
	}

	dnsService := res.(*mqlGcpProjectDnsService)
	dnsService.serviceEnabled = serviceEnabled
	dnsService.serviceChecked = true

	return dnsService, nil
}

func initGcpProjectDnsService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) peeringNetworkRef() (*mqlGcpProjectComputeServiceNetwork, error) {
	if g.PeeringNetwork.Error != nil {
		return nil, g.PeeringNetwork.Error
	}
	n, err := getNetworkByUrl(g.PeeringNetwork.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if n == nil {
		g.PeeringNetworkRef.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return n, nil
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

func (g *mqlGcpProjectDnsService) managedZones() ([]any, error) {
	// when the service is known to be disabled, we return nil
	if g.serviceChecked && !g.serviceEnabled {
		return nil, nil
	}

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

	res := []any{}
	req := dnsSvc.ManagedZones.List(projectId)
	if err := req.Pages(ctx, func(page *dns.ManagedZonesListResponse) error {
		for i := range page.ManagedZones {
			managedZone := page.ManagedZones[i]

			var mqlDnssecCfg map[string]any
			dnssecAlgorithms := []any{}
			dnssecAlgorithmSet := map[string]struct{}{}
			dnssecNonExistence := managedZoneDnssecNonExistence(managedZone.DnssecConfig)
			if managedZone.DnssecConfig != nil {
				keySpecs := make([]any, 0, len(managedZone.DnssecConfig.DefaultKeySpecs))
				for _, keySpec := range managedZone.DnssecConfig.DefaultKeySpecs {
					keySpecs = append(keySpecs, map[string]any{
						"algorithm": keySpec.Algorithm,
						"keyLength": keySpec.KeyLength,
						"keyType":   keySpec.KeyType,
					})
					// The ZSK and KSK key specs commonly share an algorithm; emit each distinct value once.
					if _, ok := dnssecAlgorithmSet[keySpec.Algorithm]; keySpec.Algorithm != "" && !ok {
						dnssecAlgorithmSet[keySpec.Algorithm] = struct{}{}
						dnssecAlgorithms = append(dnssecAlgorithms, keySpec.Algorithm)
					}
				}
				mqlDnssecCfg = map[string]any{
					"defaultKeySpecs": keySpecs,
					"nonExistence":    managedZone.DnssecConfig.NonExistence,
					"state":           managedZone.DnssecConfig.State,
				}
			}

			var mqlPrivateVisibilityCfg map[string]any
			if managedZone.PrivateVisibilityConfig != nil {
				networks := make([]any, 0, len(managedZone.PrivateVisibilityConfig.Networks))
				for _, n := range managedZone.PrivateVisibilityConfig.Networks {
					networks = append(networks, map[string]any{
						"networkUrl": n.NetworkUrl,
					})
				}
				gkeClusters := make([]any, 0, len(managedZone.PrivateVisibilityConfig.GkeClusters))
				for _, c := range managedZone.PrivateVisibilityConfig.GkeClusters {
					gkeClusters = append(gkeClusters, map[string]any{
						"gkeClusterName": c.GkeClusterName,
					})
				}
				mqlPrivateVisibilityCfg = map[string]any{
					"networks":    networks,
					"gkeClusters": gkeClusters,
				}
			}

			forwardingTargets := []any{}
			if managedZone.ForwardingConfig != nil {
				for _, target := range managedZone.ForwardingConfig.TargetNameServers {
					if target.Ipv4Address != "" {
						forwardingTargets = append(forwardingTargets, target.Ipv4Address)
					}
				}
			}

			peeringNetwork := ""
			if managedZone.PeeringConfig != nil && managedZone.PeeringConfig.TargetNetwork != nil {
				peeringNetwork = managedZone.PeeringConfig.TargetNetwork.NetworkUrl
			}

			mqlManagedZone, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService.managedzone", map[string]*llx.RawData{
				"id":                         llx.StringData(strconv.FormatInt(int64(managedZone.Id), 10)),
				"projectId":                  llx.StringData(projectId),
				"name":                       llx.StringData(managedZone.Name),
				"description":                llx.StringData(managedZone.Description),
				"dnssecConfig":               llx.DictData(mqlDnssecCfg),
				"dnsName":                    llx.StringData(managedZone.DnsName),
				"nameServerSet":              llx.StringData(managedZone.NameServerSet),
				"nameServers":                llx.ArrayData(convert.SliceAnyToInterface(managedZone.NameServers), types.String),
				"visibility":                 llx.StringData(managedZone.Visibility),
				"created":                    llx.TimeDataPtr(parseTime(managedZone.CreationTime)),
				"labels":                     llx.MapData(convert.MapToInterfaceMap(managedZone.Labels), types.String),
				"cloudLoggingEnabled":        llx.BoolData(managedZone.CloudLoggingConfig != nil && managedZone.CloudLoggingConfig.EnableLogging),
				"dnssecEnabled":              llx.BoolData(managedZone.DnssecConfig != nil && managedZone.DnssecConfig.State == "on"),
				"dnssecNonExistence":         llx.StringData(dnssecNonExistence),
				"dnssecDefaultKeyAlgorithms": llx.ArrayData(dnssecAlgorithms, types.String),
				"privateVisibilityConfig":    llx.DictData(mqlPrivateVisibilityCfg),
				"forwardingTargets":          llx.ArrayData(forwardingTargets, types.String),
				"peeringNetwork":             llx.StringData(peeringNetwork),
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

func (g *mqlGcpProjectDnsService) policies() ([]any, error) {
	// when the service is known to be disabled, we return nil
	if g.serviceChecked && !g.serviceEnabled {
		return nil, nil
	}

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

	res := []any{}
	req := dnsSvc.Policies.List(projectId)
	if err := req.Pages(ctx, func(page *dns.PoliciesListResponse) error {
		for i := range page.Policies {
			policy := page.Policies[i]

			networkNames := make([]any, 0, len(policy.Networks))
			for _, network := range policy.Networks {
				segments := strings.Split(network.NetworkUrl, "/")
				networkNames = append(networkNames, segments[len(segments)-1])
			}

			altNameServers := []any{}
			if policy.AlternativeNameServerConfig != nil {
				for _, target := range policy.AlternativeNameServerConfig.TargetNameServers {
					if target.Ipv4Address != "" {
						altNameServers = append(altNameServers, target.Ipv4Address)
					}
				}
			}

			mqlDnsPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService.policy", map[string]*llx.RawData{
				"projectId":               llx.StringData(projectId),
				"id":                      llx.StringData(strconv.FormatInt(int64(policy.Id), 10)),
				"name":                    llx.StringData(policy.Name),
				"description":             llx.StringData(policy.Description),
				"enableInboundForwarding": llx.BoolData(policy.EnableInboundForwarding),
				"enableLogging":           llx.BoolData(policy.EnableLogging),
				"networkNames":            llx.ArrayData(networkNames, types.String),
				"alternativeNameServers":  llx.ArrayData(altNameServers, types.String),
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

func (g *mqlGcpProjectDnsServiceResponsePolicy) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcp.project.dnsService.responsePolicy/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectDnsService) responsePolicies() ([]any, error) {
	// when the service is known to be disabled, we return nil
	if g.serviceChecked && !g.serviceEnabled {
		return nil, nil
	}

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

	res := []any{}
	req := dnsSvc.ResponsePolicies.List(projectId)
	if err := req.Pages(ctx, func(page *dns.ResponsePoliciesListResponse) error {
		for i := range page.ResponsePolicies {
			responsePolicy := page.ResponsePolicies[i]

			networkUrls := make([]any, 0, len(responsePolicy.Networks))
			for _, network := range responsePolicy.Networks {
				networkUrls = append(networkUrls, network.NetworkUrl)
			}

			gkeClusters := make([]any, 0, len(responsePolicy.GkeClusters))
			for _, c := range responsePolicy.GkeClusters {
				gkeClusters = append(gkeClusters, c.GkeClusterName)
			}

			mqlResponsePolicy, err := CreateResource(g.MqlRuntime, "gcp.project.dnsService.responsePolicy", map[string]*llx.RawData{
				"projectId":          llx.StringData(projectId),
				"id":                 llx.StringData(strconv.FormatInt(responsePolicy.Id, 10)),
				"responsePolicyName": llx.StringData(responsePolicy.ResponsePolicyName),
				"description":        llx.StringData(responsePolicy.Description),
				"networkUrls":        llx.ArrayData(networkUrls, types.String),
				"gkeClusters":        llx.ArrayData(gkeClusters, types.String),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlResponsePolicy)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectDnsServiceResponsePolicy) networks() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	networkUrls := g.GetNetworkUrls()
	if networkUrls.Error != nil {
		return nil, networkUrls.Error
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

	res := make([]any, 0, len(networkUrls.Data))
	for _, network := range networks.Data {
		mqlNetwork := network.(*mqlGcpProjectComputeServiceNetwork)
		for _, raw := range networkUrls.Data {
			url, ok := raw.(string)
			if !ok || url == "" {
				continue
			}
			segments := strings.Split(url, "/")
			if segments[len(segments)-1] == mqlNetwork.Name.Data {
				res = append(res, network)
				break
			}
		}
	}
	return res, nil
}

func (g *mqlGcpProjectDnsServicePolicy) networks() ([]any, error) {
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

	res := make([]any, 0, len(networkNames.Data))
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

func (g *mqlGcpProjectDnsServiceManagedzone) dnsSecAlgorithmWeak() (bool, error) {
	enabled := g.GetDnssecEnabled()
	if enabled.Error != nil {
		return false, enabled.Error
	}
	if !enabled.Data {
		return false, nil
	}
	cfg := g.GetDnssecConfig()
	if cfg.Error != nil {
		return false, cfg.Error
	}
	if cfg.Data == nil {
		return false, nil
	}
	cfgMap, ok := cfg.Data.(map[string]any)
	if !ok {
		return false, nil
	}
	specs, ok := cfgMap["defaultKeySpecs"].([]any)
	if !ok {
		return false, nil
	}
	for _, raw := range specs {
		spec, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		alg, _ := spec["algorithm"].(string)
		switch strings.ToUpper(alg) {
		case "RSASHA1", "RSASHA1-NSEC3-SHA1":
			return true, nil
		}
	}
	return false, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) iamPolicy() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	zoneName := g.Name.Data

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

	resourcePath := "projects/" + projectId + "/managedZones/" + zoneName
	policy, err := dnsSvc.ManagedZones.GetIamPolicy(resourcePath, &dns.GoogleIamV1GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(policy.Bindings))
	for i, b := range policy.Bindings {
		condTitle, condExpr, condDesc := "", "", ""
		if b.Condition != nil {
			condTitle = b.Condition.Title
			condExpr = b.Condition.Expression
			condDesc = b.Condition.Description
		}

		mqlBinding, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":                   llx.StringData(resourcePath + "-" + strconv.Itoa(i)),
			"role":                 llx.StringData(b.Role),
			"members":              llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
			"conditionTitle":       llx.StringData(condTitle),
			"conditionExpression":  llx.StringData(condExpr),
			"conditionDescription": llx.StringData(condDesc),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func (g *mqlGcpProjectDnsServiceManagedzone) recordSets() ([]any, error) {
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

	res := []any{}
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

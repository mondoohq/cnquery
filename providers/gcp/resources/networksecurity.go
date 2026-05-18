// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	networksecurity "google.golang.org/api/networksecurity/v1"
	"google.golang.org/api/option"
)

type mqlGcpProjectNetworkSecurityServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) networkSecurity() (*mqlGcpProjectNetworkSecurityService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	// Check service enablement before creating the resource: a transient error
	// here must not leave a resource cached with serviceEnabled = false, which
	// would make every child accessor silently return nil on later calls.
	serviceEnabled, err := g.isServiceEnabled(service_networksecurity)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectNetworkSecurityService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_networksecurity).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func initGcpProjectNetworkSecurityService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectNetworkSecurityService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("%s/gcp.project.networkSecurityService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectNetworkSecurityServiceAuthorizationPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectNetworkSecurityServiceServerTlsPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectNetworkSecurityServiceClientTlsPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectNetworkSecurityServiceTlsInspectionPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectNetworkSecurityServiceAddressGroup) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectNetworkSecurityServiceUrlList) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpOrganizationNetworkSecurityProfile) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpOrganizationNetworkSecurityProfileGroup) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// networkSecurityHTTPClient resolves an authenticated HTTP client for the
// networksecurity REST API. Each accessor still constructs its own
// networksecurity.Service inline so permission extraction can trace the calls.
func (g *mqlGcpProjectNetworkSecurityService) networkSecurityHTTPClient() (*connection.GcpConnection, string, error) {
	if g.ProjectId.Error != nil {
		return nil, "", g.ProjectId.Error
	}
	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, "", errors.New("invalid connection provided, it is not a GCP connection")
	}
	// locations/- aggregates resources across every location
	return conn, fmt.Sprintf("projects/%s/locations/-", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectNetworkSecurityService) authorizationPolicies() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	conn, parent, err := g.networkSecurityHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []any{}
	err = nsSvc.Projects.Locations.AuthorizationPolicies.List(parent).Pages(ctx, func(page *networksecurity.ListAuthorizationPoliciesResponse) error {
		for _, p := range page.AuthorizationPolicies {
			rules, err := convert.JsonToDictSlice(p.Rules)
			if err != nil {
				return err
			}
			mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService.authorizationPolicy", map[string]*llx.RawData{
				"name":        llx.StringData(p.Name),
				"description": llx.StringData(p.Description),
				"action":      llx.StringData(p.Action),
				"rules":       llx.ArrayData(rules, types.Dict),
				"labels":      llx.MapData(convert.MapToInterfaceMap(p.Labels), types.String),
				"created":     llx.TimeDataPtr(parseTime(p.CreateTime)),
				"updated":     llx.TimeDataPtr(parseTime(p.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlPolicy)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list authorization policies")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectNetworkSecurityService) serverTlsPolicies() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	conn, parent, err := g.networkSecurityHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []any{}
	err = nsSvc.Projects.Locations.ServerTlsPolicies.List(parent).Pages(ctx, func(page *networksecurity.ListServerTlsPoliciesResponse) error {
		for _, p := range page.ServerTlsPolicies {
			mtlsPolicy, err := convert.JsonToDict(p.MtlsPolicy)
			if err != nil {
				return err
			}
			serverCertificate, err := convert.JsonToDict(p.ServerCertificate)
			if err != nil {
				return err
			}
			mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService.serverTlsPolicy", map[string]*llx.RawData{
				"name":              llx.StringData(p.Name),
				"description":       llx.StringData(p.Description),
				"allowOpen":         llx.BoolData(p.AllowOpen),
				"mtlsPolicy":        llx.DictData(mtlsPolicy),
				"serverCertificate": llx.DictData(serverCertificate),
				"labels":            llx.MapData(convert.MapToInterfaceMap(p.Labels), types.String),
				"created":           llx.TimeDataPtr(parseTime(p.CreateTime)),
				"updated":           llx.TimeDataPtr(parseTime(p.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlPolicy)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list server TLS policies")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectNetworkSecurityService) clientTlsPolicies() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	conn, parent, err := g.networkSecurityHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []any{}
	err = nsSvc.Projects.Locations.ClientTlsPolicies.List(parent).Pages(ctx, func(page *networksecurity.ListClientTlsPoliciesResponse) error {
		for _, p := range page.ClientTlsPolicies {
			clientCertificate, err := convert.JsonToDict(p.ClientCertificate)
			if err != nil {
				return err
			}
			serverValidationCa, err := convert.JsonToDictSlice(p.ServerValidationCa)
			if err != nil {
				return err
			}
			mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService.clientTlsPolicy", map[string]*llx.RawData{
				"name":               llx.StringData(p.Name),
				"description":        llx.StringData(p.Description),
				"sni":                llx.StringData(p.Sni),
				"clientCertificate":  llx.DictData(clientCertificate),
				"serverValidationCa": llx.ArrayData(serverValidationCa, types.Dict),
				"labels":             llx.MapData(convert.MapToInterfaceMap(p.Labels), types.String),
				"created":            llx.TimeDataPtr(parseTime(p.CreateTime)),
				"updated":            llx.TimeDataPtr(parseTime(p.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlPolicy)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list client TLS policies")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectNetworkSecurityService) tlsInspectionPolicies() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	conn, parent, err := g.networkSecurityHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []any{}
	err = nsSvc.Projects.Locations.TlsInspectionPolicies.List(parent).Pages(ctx, func(page *networksecurity.ListTlsInspectionPoliciesResponse) error {
		for _, p := range page.TlsInspectionPolicies {
			customTlsFeatures := make([]any, 0, len(p.CustomTlsFeatures))
			for _, f := range p.CustomTlsFeatures {
				customTlsFeatures = append(customTlsFeatures, f)
			}
			mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService.tlsInspectionPolicy", map[string]*llx.RawData{
				"name":               llx.StringData(p.Name),
				"description":        llx.StringData(p.Description),
				"caPool":             llx.StringData(p.CaPool),
				"minTlsVersion":      llx.StringData(p.MinTlsVersion),
				"tlsFeatureProfile":  llx.StringData(p.TlsFeatureProfile),
				"customTlsFeatures":  llx.ArrayData(customTlsFeatures, types.String),
				"excludePublicCaSet": llx.BoolData(p.ExcludePublicCaSet),
				"trustConfig":        llx.StringData(p.TrustConfig),
				"created":            llx.TimeDataPtr(parseTime(p.CreateTime)),
				"updated":            llx.TimeDataPtr(parseTime(p.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlPolicy)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list TLS inspection policies")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectNetworkSecurityService) addressGroups() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	conn, parent, err := g.networkSecurityHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []any{}
	err = nsSvc.Projects.Locations.AddressGroups.List(parent).Pages(ctx, func(page *networksecurity.ListAddressGroupsResponse) error {
		for _, ag := range page.AddressGroups {
			items := make([]any, 0, len(ag.Items))
			for _, item := range ag.Items {
				items = append(items, item)
			}
			purpose := make([]any, 0, len(ag.Purpose))
			for _, p := range ag.Purpose {
				purpose = append(purpose, p)
			}
			mqlGroup, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService.addressGroup", map[string]*llx.RawData{
				"name":        llx.StringData(ag.Name),
				"description": llx.StringData(ag.Description),
				"type":        llx.StringData(ag.Type),
				"items":       llx.ArrayData(items, types.String),
				"capacity":    llx.IntData(ag.Capacity),
				"purpose":     llx.ArrayData(purpose, types.String),
				"labels":      llx.MapData(convert.MapToInterfaceMap(ag.Labels), types.String),
				"created":     llx.TimeDataPtr(parseTime(ag.CreateTime)),
				"updated":     llx.TimeDataPtr(parseTime(ag.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlGroup)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list address groups")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectNetworkSecurityService) urlLists() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	conn, parent, err := g.networkSecurityHTTPClient()
	if err != nil {
		return nil, err
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []any{}
	err = nsSvc.Projects.Locations.UrlLists.List(parent).Pages(ctx, func(page *networksecurity.ListUrlListsResponse) error {
		for _, ul := range page.UrlLists {
			values := make([]any, 0, len(ul.Values))
			for _, v := range ul.Values {
				values = append(values, v)
			}
			mqlUrlList, err := CreateResource(g.MqlRuntime, "gcp.project.networkSecurityService.urlList", map[string]*llx.RawData{
				"name":        llx.StringData(ul.Name),
				"description": llx.StringData(ul.Description),
				"values":      llx.ArrayData(values, types.String),
				"created":     llx.TimeDataPtr(parseTime(ul.CreateTime)),
				"updated":     llx.TimeDataPtr(parseTime(ul.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlUrlList)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list URL lists")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

// networkSecurityProfiles lists organization-scoped network security profiles.
// Organization-scoped accessors have no serviceusage service-enabled pre-check
// (enablement is per-project); a disabled API surfaces as a 403/404 that
// isHTTPSkippable handles below.
func (g *mqlGcpOrganization) networkSecurityProfiles() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("%s/locations/-", g.Id.Data)
	res := []any{}
	err = nsSvc.Organizations.Locations.SecurityProfiles.List(parent).Pages(ctx, func(page *networksecurity.ListSecurityProfilesResponse) error {
		for _, sp := range page.SecurityProfiles {
			threatPreventionProfile, err := convert.JsonToDict(sp.ThreatPreventionProfile)
			if err != nil {
				return err
			}
			urlFilteringProfile, err := convert.JsonToDict(sp.UrlFilteringProfile)
			if err != nil {
				return err
			}
			customMirroringProfile, err := convert.JsonToDict(sp.CustomMirroringProfile)
			if err != nil {
				return err
			}
			customInterceptProfile, err := convert.JsonToDict(sp.CustomInterceptProfile)
			if err != nil {
				return err
			}
			mqlProfile, err := CreateResource(g.MqlRuntime, "gcp.organization.networkSecurityProfile", map[string]*llx.RawData{
				"name":                    llx.StringData(sp.Name),
				"description":             llx.StringData(sp.Description),
				"type":                    llx.StringData(sp.Type),
				"threatPreventionProfile": llx.DictData(threatPreventionProfile),
				"urlFilteringProfile":     llx.DictData(urlFilteringProfile),
				"customMirroringProfile":  llx.DictData(customMirroringProfile),
				"customInterceptProfile":  llx.DictData(customInterceptProfile),
				"labels":                  llx.MapData(convert.MapToInterfaceMap(sp.Labels), types.String),
				"etag":                    llx.StringData(sp.Etag),
				"created":                 llx.TimeDataPtr(parseTime(sp.CreateTime)),
				"updated":                 llx.TimeDataPtr(parseTime(sp.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlProfile)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list network security profiles")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

// networkSecurityProfileGroups lists organization-scoped network security
// profile groups. See networkSecurityProfiles for why there is no
// service-enabled pre-check.
func (g *mqlGcpOrganization) networkSecurityProfileGroups() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(networksecurity.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	nsSvc, err := networksecurity.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("%s/locations/-", g.Id.Data)
	res := []any{}
	err = nsSvc.Organizations.Locations.SecurityProfileGroups.List(parent).Pages(ctx, func(page *networksecurity.ListSecurityProfileGroupsResponse) error {
		for _, spg := range page.SecurityProfileGroups {
			mqlGroup, err := CreateResource(g.MqlRuntime, "gcp.organization.networkSecurityProfileGroup", map[string]*llx.RawData{
				"name":                    llx.StringData(spg.Name),
				"description":             llx.StringData(spg.Description),
				"threatPreventionProfile": llx.StringData(spg.ThreatPreventionProfile),
				"customMirroringProfile":  llx.StringData(spg.CustomMirroringProfile),
				"customInterceptProfile":  llx.StringData(spg.CustomInterceptProfile),
				"labels":                  llx.MapData(convert.MapToInterfaceMap(spg.Labels), types.String),
				"etag":                    llx.StringData(spg.Etag),
				"created":                 llx.TimeDataPtr(parseTime(spg.CreateTime)),
				"updated":                 llx.TimeDataPtr(parseTime(spg.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlGroup)
		}
		return nil
	})
	if err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list network security profile groups")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cdn/armcdn/v2"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionFrontDoorService) id() (string, error) {
	return "azure.subscription.frontDoor/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionFrontDoorService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())
	return args, nil, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfile) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfileEndpoint) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfileCustomDomain) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfileOriginGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfileOriginGroupOrigin) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFrontDoorService) profiles() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armcdn.NewProfilesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, profile := range page.Value {
			if profile == nil {
				continue
			}

			sku, err := convert.JsonToDict(profile.SKU)
			if err != nil {
				return nil, err
			}

			var provisioningState, frontDoorId, resourceState string
			if profile.Properties != nil {
				if profile.Properties.ProvisioningState != nil {
					provisioningState = string(*profile.Properties.ProvisioningState)
				}
				if profile.Properties.FrontDoorID != nil {
					frontDoorId = *profile.Properties.FrontDoorID
				}
				if profile.Properties.ResourceState != nil {
					resourceState = string(*profile.Properties.ResourceState)
				}
			}

			mqlProfile, err := CreateResource(a.MqlRuntime, "azure.subscription.frontDoorService.profile", map[string]*llx.RawData{
				"id":                llx.StringDataPtr(profile.ID),
				"name":              llx.StringDataPtr(profile.Name),
				"location":          llx.StringDataPtr(profile.Location),
				"tags":              llx.MapData(convert.PtrMapStrToInterface(profile.Tags), types.String),
				"sku":               llx.DictData(sku),
				"kind":              llx.StringDataPtr(profile.Kind),
				"provisioningState": llx.StringData(provisioningState),
				"frontDoorId":       llx.StringData(frontDoorId),
				"resourceState":     llx.StringData(resourceState),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlProfile)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfile) endpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	profileName := a.Name.Data

	client, err := armcdn.NewAFDEndpointsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByProfilePager(resourceID.ResourceGroup, profileName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ep := range page.Value {
			if ep == nil {
				continue
			}

			var hostName, enabledState, provisioningState string
			if ep.Properties != nil {
				if ep.Properties.HostName != nil {
					hostName = *ep.Properties.HostName
				}
				if ep.Properties.EnabledState != nil {
					enabledState = string(*ep.Properties.EnabledState)
				}
				if ep.Properties.ProvisioningState != nil {
					provisioningState = string(*ep.Properties.ProvisioningState)
				}
			}

			mqlEp, err := CreateResource(a.MqlRuntime, "azure.subscription.frontDoorService.profile.endpoint", map[string]*llx.RawData{
				"id":                llx.StringDataPtr(ep.ID),
				"name":              llx.StringDataPtr(ep.Name),
				"location":          llx.StringDataPtr(ep.Location),
				"hostName":          llx.StringData(hostName),
				"enabledState":      llx.StringData(enabledState),
				"provisioningState": llx.StringData(provisioningState),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEp)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfile) customDomains() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	profileName := a.Name.Data

	client, err := armcdn.NewAFDCustomDomainsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByProfilePager(resourceID.ResourceGroup, profileName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cd := range page.Value {
			if cd == nil {
				continue
			}

			var hostName, validationState, provisioningState, tlsMinimumVersion, tlsCertificateType string
			if cd.Properties != nil {
				if cd.Properties.HostName != nil {
					hostName = *cd.Properties.HostName
				}
				if cd.Properties.DomainValidationState != nil {
					validationState = string(*cd.Properties.DomainValidationState)
				}
				if cd.Properties.ProvisioningState != nil {
					provisioningState = string(*cd.Properties.ProvisioningState)
				}
				if cd.Properties.TLSSettings != nil {
					if cd.Properties.TLSSettings.MinimumTLSVersion != nil {
						tlsMinimumVersion = string(*cd.Properties.TLSSettings.MinimumTLSVersion)
					}
					if cd.Properties.TLSSettings.CertificateType != nil {
						tlsCertificateType = string(*cd.Properties.TLSSettings.CertificateType)
					}
				}
			}

			mqlCd, err := CreateResource(a.MqlRuntime, "azure.subscription.frontDoorService.profile.customDomain", map[string]*llx.RawData{
				"id":                 llx.StringDataPtr(cd.ID),
				"name":               llx.StringDataPtr(cd.Name),
				"hostName":           llx.StringData(hostName),
				"validationState":    llx.StringData(validationState),
				"provisioningState":  llx.StringData(provisioningState),
				"tlsMinimumVersion":  llx.StringData(tlsMinimumVersion),
				"tlsCertificateType": llx.StringData(tlsCertificateType),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCd)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfile) originGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	profileName := a.Name.Data

	client, err := armcdn.NewAFDOriginGroupsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByProfilePager(resourceID.ResourceGroup, profileName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, og := range page.Value {
			if og == nil {
				continue
			}

			var provisioningState string
			var healthProbeSettings, loadBalancingSettings any
			if og.Properties != nil {
				if og.Properties.ProvisioningState != nil {
					provisioningState = string(*og.Properties.ProvisioningState)
				}
				var err error
				healthProbeSettings, err = convert.JsonToDict(og.Properties.HealthProbeSettings)
				if err != nil {
					return nil, err
				}
				loadBalancingSettings, err = convert.JsonToDict(og.Properties.LoadBalancingSettings)
				if err != nil {
					return nil, err
				}
			}

			mqlOg, err := CreateResource(a.MqlRuntime, "azure.subscription.frontDoorService.profile.originGroup", map[string]*llx.RawData{
				"id":                    llx.StringDataPtr(og.ID),
				"name":                  llx.StringDataPtr(og.Name),
				"provisioningState":     llx.StringData(provisioningState),
				"healthProbeSettings":   llx.DictData(healthProbeSettings),
				"loadBalancingSettings": llx.DictData(loadBalancingSettings),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlOg)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfileOriginGroup) origins() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	profileName, err := resourceID.Component("profiles")
	if err != nil {
		return nil, err
	}

	originGroupName := a.Name.Data

	client, err := armcdn.NewAFDOriginsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByOriginGroupPager(resourceID.ResourceGroup, profileName, originGroupName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, origin := range page.Value {
			if origin == nil {
				continue
			}

			var hostName, originHostHeader, enabledState, provisioningState string
			var httpPort, httpsPort, priority, weight int64
			if origin.Properties != nil {
				if origin.Properties.HostName != nil {
					hostName = *origin.Properties.HostName
				}
				if origin.Properties.OriginHostHeader != nil {
					originHostHeader = *origin.Properties.OriginHostHeader
				}
				if origin.Properties.EnabledState != nil {
					enabledState = string(*origin.Properties.EnabledState)
				}
				if origin.Properties.ProvisioningState != nil {
					provisioningState = string(*origin.Properties.ProvisioningState)
				}
				if origin.Properties.HTTPPort != nil {
					httpPort = int64(*origin.Properties.HTTPPort)
				}
				if origin.Properties.HTTPSPort != nil {
					httpsPort = int64(*origin.Properties.HTTPSPort)
				}
				if origin.Properties.Priority != nil {
					priority = int64(*origin.Properties.Priority)
				}
				if origin.Properties.Weight != nil {
					weight = int64(*origin.Properties.Weight)
				}
			}

			mqlOrigin, err := CreateResource(a.MqlRuntime, "azure.subscription.frontDoorService.profile.originGroup.origin", map[string]*llx.RawData{
				"id":                llx.StringDataPtr(origin.ID),
				"name":              llx.StringDataPtr(origin.Name),
				"hostName":          llx.StringData(hostName),
				"httpPort":          llx.IntData(httpPort),
				"httpsPort":         llx.IntData(httpsPort),
				"originHostHeader":  llx.StringData(originHostHeader),
				"priority":          llx.IntData(priority),
				"weight":            llx.IntData(weight),
				"enabledState":      llx.StringData(enabledState),
				"provisioningState": llx.StringData(provisioningState),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlOrigin)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionFrontDoorServiceProfileEndpointRoute) id() (string, error) {
	return a.Id.Data, nil
}

// routes fetches the routes for a Front Door endpoint.
func (a *mqlAzureSubscriptionFrontDoorServiceProfileEndpoint) routes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	profileName, err := resourceID.Component("profiles")
	if err != nil {
		return nil, err
	}
	endpointName, err := resourceID.Component("afdEndpoints")
	if err != nil {
		return nil, err
	}

	client, err := armcdn.NewRoutesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByEndpointPager(resourceID.ResourceGroup, profileName, endpointName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rt := range page.Value {
			if rt == nil {
				continue
			}
			var httpsRedirect, forwardingProtocol, linkToDefault, enabledState, originPath, originGroupId, provisioningState string
			var supportedProtocols, patternsToMatch []any
			if rt.Properties != nil {
				if rt.Properties.HTTPSRedirect != nil {
					httpsRedirect = string(*rt.Properties.HTTPSRedirect)
				}
				if rt.Properties.ForwardingProtocol != nil {
					forwardingProtocol = string(*rt.Properties.ForwardingProtocol)
				}
				if rt.Properties.LinkToDefaultDomain != nil {
					linkToDefault = string(*rt.Properties.LinkToDefaultDomain)
				}
				if rt.Properties.EnabledState != nil {
					enabledState = string(*rt.Properties.EnabledState)
				}
				if rt.Properties.OriginPath != nil {
					originPath = *rt.Properties.OriginPath
				}
				if rt.Properties.OriginGroup != nil && rt.Properties.OriginGroup.ID != nil {
					originGroupId = *rt.Properties.OriginGroup.ID
				}
				if rt.Properties.ProvisioningState != nil {
					provisioningState = string(*rt.Properties.ProvisioningState)
				}
				for _, p := range rt.Properties.SupportedProtocols {
					if p != nil {
						supportedProtocols = append(supportedProtocols, string(*p))
					}
				}
				for _, p := range rt.Properties.PatternsToMatch {
					if p != nil {
						patternsToMatch = append(patternsToMatch, *p)
					}
				}
			}
			mqlRt, err := CreateResource(a.MqlRuntime, "azure.subscription.frontDoorService.profile.endpoint.route", map[string]*llx.RawData{
				"id":                  llx.StringDataPtr(rt.ID),
				"name":                llx.StringDataPtr(rt.Name),
				"type":                llx.StringDataPtr(rt.Type),
				"httpsRedirect":       llx.StringData(httpsRedirect),
				"forwardingProtocol":  llx.StringData(forwardingProtocol),
				"linkToDefaultDomain": llx.StringData(linkToDefault),
				"enabledState":        llx.StringData(enabledState),
				"supportedProtocols":  llx.ArrayData(supportedProtocols, types.String),
				"patternsToMatch":     llx.ArrayData(patternsToMatch, types.String),
				"originPath":          llx.StringData(originPath),
				"originGroupId":       llx.StringData(originGroupId),
				"provisioningState":   llx.StringData(provisioningState),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRt)
		}
	}
	return res, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/iothub/armiothub"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionIotServiceIotHub) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionIotServiceIotHub) diagnosticSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings(a.Id.Data, a.MqlRuntime, conn)
}

func initAzureSubscriptionIotService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, fmt.Errorf("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionIotService) hubs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	subscriptionID := a.SubscriptionId.Data

	clientFactory, err := armiothub.NewClientFactory(subscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewResourceClient()
	hubsPager := client.NewListBySubscriptionPager(nil)
	var hubs []any

	for hubsPager.More() {
		page, err := hubsPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, hub := range page.Value {
			hubData, err := convert.JsonToDict(hub)
			if err != nil {
				return nil, err
			}
			hubs = append(hubs, hubData)
		}
	}

	return hubs, nil
}

// iotHubs returns typed IoT hub resources with security-relevant fields populated.
func (a *mqlAzureSubscriptionIotService) iotHubs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subscriptionID := a.SubscriptionId.Data

	clientFactory, err := armiothub.NewClientFactory(subscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	client := clientFactory.NewResourceClient()
	pager := client.NewListBySubscriptionPager(nil)

	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, hub := range page.Value {
			if hub == nil {
				continue
			}

			sku, err := convert.JsonToDict(hub.SKU)
			if err != nil {
				return nil, err
			}

			var provisioningState, state, hostName, minTlsVersion, publicNetworkAccess string
			var disableLocalAuth, disableDeviceSAS, disableModuleSAS, restrictOutbound, enableDataResidency *bool
			var allowedFqdns []any
			var nrs any
			if props := hub.Properties; props != nil {
				if props.ProvisioningState != nil {
					provisioningState = *props.ProvisioningState
				}
				if props.State != nil {
					state = *props.State
				}
				if props.HostName != nil {
					hostName = *props.HostName
				}
				if props.MinTLSVersion != nil {
					minTlsVersion = *props.MinTLSVersion
				}
				if props.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*props.PublicNetworkAccess)
				}
				disableLocalAuth = props.DisableLocalAuth
				disableDeviceSAS = props.DisableDeviceSAS
				disableModuleSAS = props.DisableModuleSAS
				restrictOutbound = props.RestrictOutboundNetworkAccess
				enableDataResidency = props.EnableDataResidency
				for _, fqdn := range props.AllowedFqdnList {
					if fqdn != nil {
						allowedFqdns = append(allowedFqdns, *fqdn)
					}
				}
				if props.NetworkRuleSets != nil {
					if d, err := convert.JsonToDict(props.NetworkRuleSets); err == nil {
						nrs = d
					}
				}
			}

			mqlHub, err := CreateResource(a.MqlRuntime, "azure.subscription.iotService.iotHub", map[string]*llx.RawData{
				"id":                            llx.StringDataPtr(hub.ID),
				"name":                          llx.StringDataPtr(hub.Name),
				"type":                          llx.StringDataPtr(hub.Type),
				"location":                      llx.StringDataPtr(hub.Location),
				"tags":                          llx.MapData(convert.PtrMapStrToInterface(hub.Tags), types.String),
				"sku":                           llx.DictData(sku),
				"provisioningState":             llx.StringData(provisioningState),
				"state":                         llx.StringData(state),
				"hostName":                      llx.StringData(hostName),
				"disableLocalAuth":              llx.BoolDataPtr(disableLocalAuth),
				"disableDeviceSAS":              llx.BoolDataPtr(disableDeviceSAS),
				"disableModuleSAS":              llx.BoolDataPtr(disableModuleSAS),
				"minTlsVersion":                 llx.StringData(minTlsVersion),
				"publicNetworkAccess":           llx.StringData(publicNetworkAccess),
				"restrictOutboundNetworkAccess": llx.BoolDataPtr(restrictOutbound),
				"allowedFqdnList":               llx.ArrayData(allowedFqdns, types.String),
				"enableDataResidency":           llx.BoolDataPtr(enableDataResidency),
				"networkRuleSet":                llx.DictData(nrs),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlHub)
		}
	}
	return res, nil
}

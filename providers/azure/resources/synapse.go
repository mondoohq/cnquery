// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionSynapseService) id() (string, error) {
	return "azure.subscription.synapse/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionSynapseService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionSynapseService) workspaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId := a.SubscriptionId.Data
	ctx := context.Background()

	client, err := armsynapse.NewWorkspacesClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list synapse workspaces due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, ws := range page.Value {
			if ws == nil {
				continue
			}

			properties, err := convert.JsonToDict(ws.Properties)
			if err != nil {
				return nil, err
			}
			identity, err := convert.JsonToDict(ws.Identity)
			if err != nil {
				return nil, err
			}

			var managedVirtualNetwork string
			var publicNetworkAccess string
			var encryption any
			var managedResourceGroupName string
			var sqlAdministratorLogin string
			var provisioningState string
			var trustedServiceBypassEnabled *bool
			var azureADOnlyAuthentication *bool

			if ws.Properties != nil {
				if ws.Properties.ManagedVirtualNetwork != nil {
					managedVirtualNetwork = *ws.Properties.ManagedVirtualNetwork
				}
				if ws.Properties.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*ws.Properties.PublicNetworkAccess)
				}
				if ws.Properties.Encryption != nil {
					encryption, err = convert.JsonToDict(ws.Properties.Encryption)
					if err != nil {
						return nil, err
					}
				}
				if ws.Properties.ManagedResourceGroupName != nil {
					managedResourceGroupName = *ws.Properties.ManagedResourceGroupName
				}
				if ws.Properties.SQLAdministratorLogin != nil {
					sqlAdministratorLogin = *ws.Properties.SQLAdministratorLogin
				}
				if ws.Properties.ProvisioningState != nil {
					provisioningState = *ws.Properties.ProvisioningState
				}
				trustedServiceBypassEnabled = ws.Properties.TrustedServiceBypassEnabled
				azureADOnlyAuthentication = ws.Properties.AzureADOnlyAuthentication
			}

			mqlWorkspace, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionSynapseServiceWorkspace,
				map[string]*llx.RawData{
					"__id":                        llx.StringDataPtr(ws.ID),
					"id":                          llx.StringDataPtr(ws.ID),
					"name":                        llx.StringDataPtr(ws.Name),
					"location":                    llx.StringDataPtr(ws.Location),
					"tags":                        llx.MapData(convert.PtrMapStrToInterface(ws.Tags), types.String),
					"type":                        llx.StringDataPtr(ws.Type),
					"properties":                  llx.DictData(properties),
					"identity":                    llx.DictData(identity),
					"managedVirtualNetwork":       llx.StringData(managedVirtualNetwork),
					"publicNetworkAccess":         llx.StringData(publicNetworkAccess),
					"encryption":                  llx.DictData(encryption),
					"managedResourceGroupName":    llx.StringData(managedResourceGroupName),
					"sqlAdministratorLogin":       llx.StringData(sqlAdministratorLogin),
					"provisioningState":           llx.StringData(provisioningState),
					"trustedServiceBypassEnabled": llx.BoolDataPtr(trustedServiceBypassEnabled),
					"azureADOnlyAuthentication":   llx.BoolDataPtr(azureADOnlyAuthentication),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWorkspace)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionSynapseServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

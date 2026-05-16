// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionFunctionsService) id() (string, error) {
	return "azure.subscription.functions/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionFunctionsService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionFunctionsServiceFunctionApp) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFunctionsServiceFunctionAppFunction) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionFunctionsService) functionApps() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := web.NewWebAppsClient(subId, token, &arm.ClientOptions{
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
		for _, site := range page.Value {
			if site == nil {
				continue
			}
			// Filter for function apps only
			if site.Kind == nil || !strings.Contains(strings.ToLower(*site.Kind), "functionapp") {
				continue
			}

			properties, err := convert.JsonToDict(site.Properties)
			if err != nil {
				return nil, err
			}

			var state, defaultHostName, clientCertMode, managedServiceIdentityId string
			var keyVaultReferenceIdentity string
			var httpsOnly, clientCertEnabled bool
			if site.Properties != nil {
				if site.Properties.State != nil {
					state = *site.Properties.State
				}
				if site.Properties.DefaultHostName != nil {
					defaultHostName = *site.Properties.DefaultHostName
				}
				if site.Properties.HTTPSOnly != nil {
					httpsOnly = *site.Properties.HTTPSOnly
				}
				if site.Properties.ClientCertEnabled != nil {
					clientCertEnabled = *site.Properties.ClientCertEnabled
				}
				if site.Properties.ClientCertMode != nil {
					clientCertMode = string(*site.Properties.ClientCertMode)
				}
				if site.Properties.KeyVaultReferenceIdentity != nil {
					keyVaultReferenceIdentity = *site.Properties.KeyVaultReferenceIdentity
				}
			}
			if site.Identity != nil && site.Identity.PrincipalID != nil {
				managedServiceIdentityId = *site.Identity.PrincipalID
			}

			mqlApp, err := CreateResource(a.MqlRuntime, "azure.subscription.functionsService.functionApp", map[string]*llx.RawData{
				"id":                        llx.StringDataPtr(site.ID),
				"name":                      llx.StringDataPtr(site.Name),
				"location":                  llx.StringDataPtr(site.Location),
				"tags":                      llx.MapData(convert.PtrMapStrToInterface(site.Tags), types.String),
				"kind":                      llx.StringDataPtr(site.Kind),
				"state":                     llx.StringData(state),
				"defaultHostName":           llx.StringData(defaultHostName),
				"httpsOnly":                 llx.BoolData(httpsOnly),
				"clientCertEnabled":         llx.BoolData(clientCertEnabled),
				"clientCertMode":            llx.StringData(clientCertMode),
				"managedServiceIdentityId":  llx.StringData(managedServiceIdentityId),
				"keyVaultReferenceIdentity": llx.StringData(keyVaultReferenceIdentity),
				"properties":                llx.DictData(properties),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlApp)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionFunctionsServiceFunctionApp) functions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	appName := a.Name.Data

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListFunctionsPager(resourceID.ResourceGroup, appName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, fn := range page.Value {
			if fn == nil {
				continue
			}

			var config any
			var language string
			var isDisabled bool
			if fn.Properties != nil {
				config, err = convert.JsonToDict(fn.Properties.Config)
				if err != nil {
					return nil, err
				}
				if fn.Properties.Language != nil {
					language = *fn.Properties.Language
				}
				if fn.Properties.IsDisabled != nil {
					isDisabled = *fn.Properties.IsDisabled
				}
			}

			mqlFn, err := CreateResource(a.MqlRuntime, "azure.subscription.functionsService.functionApp.function", map[string]*llx.RawData{
				"id":         llx.StringDataPtr(fn.ID),
				"name":       llx.StringDataPtr(fn.Name),
				"config":     llx.DictData(config),
				"language":   llx.StringData(language),
				"isDisabled": llx.BoolData(isDisabled),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFn)
		}
	}

	return res, nil
}

// configuration returns the function app's site configuration (minimum TLS
// version, FTPS state, HTTP/2, always-on, IP restrictions). Function apps are
// App Service sites, so this reuses the appsiteconfig resource.
func (a *mqlAzureSubscriptionFunctionsServiceFunctionApp) configuration() (*mqlAzureSubscriptionWebServiceAppsiteconfig, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return webAppSiteConfigToMql(a.MqlRuntime, conn, a.Id.Data)
}

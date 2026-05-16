// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v2"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionCognitiveServicesService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionCognitiveServicesService) id() (string, error) {
	return "azure.subscription.cognitiveServicesService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesService) accounts() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := armcognitiveservices.NewAccountsClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list cognitive services accounts due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, account := range page.Value {
			if account == nil {
				continue
			}
			mqlAccount, err := cognitiveServicesAccountToMql(a.MqlRuntime, account)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAccount)
		}
	}
	return res, nil
}

func cognitiveServicesAccountToMql(runtime *plugin.Runtime, account *armcognitiveservices.Account) (*mqlAzureSubscriptionCognitiveServicesServiceAccount, error) {
	sku, err := convert.JsonToDict(account.SKU)
	if err != nil {
		return nil, err
	}
	identity, err := convert.JsonToDict(account.Identity)
	if err != nil {
		return nil, err
	}

	var publicNetworkAccess, customSubDomainName string
	var cmkKeySource, cmkKeyName, cmkKeyVaultUri string
	var endpoint, provisioningState string
	var disableLocalAuth, restrictOutboundNetworkAccess bool
	networkAcls := llx.NilData

	if p := account.Properties; p != nil {
		if p.PublicNetworkAccess != nil {
			publicNetworkAccess = string(*p.PublicNetworkAccess)
		}
		if p.DisableLocalAuth != nil {
			disableLocalAuth = *p.DisableLocalAuth
		}
		if p.RestrictOutboundNetworkAccess != nil {
			restrictOutboundNetworkAccess = *p.RestrictOutboundNetworkAccess
		}
		if p.CustomSubDomainName != nil {
			customSubDomainName = *p.CustomSubDomainName
		}
		if p.NetworkACLs != nil {
			d, err := convert.JsonToDict(p.NetworkACLs)
			if err != nil {
				return nil, err
			}
			networkAcls = llx.DictData(d)
		}
		if enc := p.Encryption; enc != nil {
			if enc.KeySource != nil {
				cmkKeySource = string(*enc.KeySource)
			}
			if enc.KeyVaultProperties != nil {
				if enc.KeyVaultProperties.KeyName != nil {
					cmkKeyName = *enc.KeyVaultProperties.KeyName
				}
				if enc.KeyVaultProperties.KeyVaultURI != nil {
					cmkKeyVaultUri = *enc.KeyVaultProperties.KeyVaultURI
				}
			}
		}
		if p.Endpoint != nil {
			endpoint = *p.Endpoint
		}
		if p.ProvisioningState != nil {
			provisioningState = string(*p.ProvisioningState)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.cognitiveServicesService.account", map[string]*llx.RawData{
		"id":                            llx.StringDataPtr(account.ID),
		"name":                          llx.StringDataPtr(account.Name),
		"location":                      llx.StringDataPtr(account.Location),
		"kind":                          llx.StringDataPtr(account.Kind),
		"tags":                          llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
		"sku":                           llx.DictData(sku),
		"identity":                      llx.DictData(identity),
		"publicNetworkAccess":           llx.StringData(publicNetworkAccess),
		"disableLocalAuth":              llx.BoolData(disableLocalAuth),
		"restrictOutboundNetworkAccess": llx.BoolData(restrictOutboundNetworkAccess),
		"customSubDomainName":           llx.StringData(customSubDomainName),
		"networkAcls":                   networkAcls,
		"cmkKeySource":                  llx.StringData(cmkKeySource),
		"cmkKeyName":                    llx.StringData(cmkKeyName),
		"cmkKeyVaultUri":                llx.StringData(cmkKeyVaultUri),
		"endpoint":                      llx.StringData(endpoint),
		"provisioningState":             llx.StringData(provisioningState),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionCognitiveServicesServiceAccount), nil
}

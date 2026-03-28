// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datafactory/armdatafactory/v9"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionDataFactoryService) id() (string, error) {
	return "azure.subscription.dataFactory/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionDataFactoryService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionDataFactoryService) factories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId := a.SubscriptionId.Data
	ctx := context.Background()

	client, err := armdatafactory.NewFactoriesClient(subId, conn.Token(), &arm.ClientOptions{
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
				log.Warn().Err(err).Msg("could not list data factories due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, factory := range page.Value {
			if factory == nil {
				continue
			}

			properties, err := convert.JsonToDict(factory.Properties)
			if err != nil {
				return nil, err
			}
			identity, err := convert.JsonToDict(factory.Identity)
			if err != nil {
				return nil, err
			}

			var publicNetworkAccess string
			var provisioningState string
			var version string
			var repoConfig any
			var encryption any
			var created *llx.RawData

			if factory.Properties != nil {
				if factory.Properties.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*factory.Properties.PublicNetworkAccess)
				}
				if factory.Properties.ProvisioningState != nil {
					provisioningState = *factory.Properties.ProvisioningState
				}
				if factory.Properties.Version != nil {
					version = *factory.Properties.Version
				}
				if factory.Properties.RepoConfiguration != nil {
					repoConfig, err = convert.JsonToDict(factory.Properties.RepoConfiguration)
					if err != nil {
						return nil, err
					}
				}
				if factory.Properties.Encryption != nil {
					encryption, err = convert.JsonToDict(factory.Properties.Encryption)
					if err != nil {
						return nil, err
					}
				}
				created = llx.TimeDataPtr(factory.Properties.CreateTime)
			}
			if created == nil {
				created = llx.TimeData(llx.NeverFutureTime)
			}

			mqlFactory, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionDataFactoryServiceFactory,
				map[string]*llx.RawData{
					"__id":                llx.StringDataPtr(factory.ID),
					"id":                  llx.StringDataPtr(factory.ID),
					"name":                llx.StringDataPtr(factory.Name),
					"location":            llx.StringDataPtr(factory.Location),
					"tags":                llx.MapData(convert.PtrMapStrToInterface(factory.Tags), types.String),
					"type":                llx.StringDataPtr(factory.Type),
					"properties":          llx.DictData(properties),
					"publicNetworkAccess": llx.StringData(publicNetworkAccess),
					"identity":            llx.DictData(identity),
					"provisioningState":   llx.StringData(provisioningState),
					"version":             llx.StringData(version),
					"repoConfiguration":   llx.DictData(repoConfig),
					"encryption":          llx.DictData(encryption),
					"created":             created,
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFactory)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionDataFactoryServiceFactory) id() (string, error) {
	return a.Id.Data, nil
}

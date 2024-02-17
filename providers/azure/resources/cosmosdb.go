// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	cosmosdb "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
)

func (a *mqlAzureSubscriptionCosmosDbService) id() (string, error) {
	return "azure.subscription.cosmosdb/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionCosmosDbService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCosmosDbService) accounts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	accClient, err := cosmosdb.NewDatabaseAccountsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := accClient.NewListPager(&cosmosdb.DatabaseAccountsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, account := range page.Value {
			properties, err := convert.JsonToDict(account.Properties)
			if err != nil {
				return nil, err
			}

			mqlCosmosDbAccount, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(account.ID),
					"name":       llx.StringDataPtr(account.Name),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
					"location":   llx.StringDataPtr(account.Location),
					"kind":       llx.StringDataPtr((*string)(account.Kind)),
					"type":       llx.StringDataPtr(account.Type),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCosmosDbAccount)
		}
	}
	return res, nil
}

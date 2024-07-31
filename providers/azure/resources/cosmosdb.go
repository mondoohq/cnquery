// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	cosmosdb "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	armresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func (a *mqlAzureSubscriptionCosmosDbService) id() (string, error) {
	return "azure.subscription.cosmosdb/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionCosmosDbService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCosmosDbService) accounts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	res := []interface{}{}

	// Fetch resources of different types - other than MongoDB and PostgreSQL
	cosmosAccounts, err := fetchCosmosDBAccounts(ctx, a.MqlRuntime, conn, subId)
	if err != nil {
		return nil, err
	}
	res = append(res, cosmosAccounts...)

	mongoAccounts, err := fetchDbAccountsByType(ctx, a.MqlRuntime, conn, subId, "Microsoft.DocumentDB/mongoClusters")
	if err != nil {
		return nil, err
	}
	res = append(res, mongoAccounts...)

	postgresAccounts, err := fetchDbAccountsByType(ctx, a.MqlRuntime, conn, subId, "Microsoft.DBforPostgreSQL/serverGroupsv2")
	if err != nil {
		return nil, err
	}
	res = append(res, postgresAccounts...)

	return res, nil
}

func fetchCosmosDBAccounts(ctx context.Context, runtime *plugin.Runtime, conn *connection.AzureConnection, subId string) ([]interface{}, error) {
	accClient, err := cosmosdb.NewDatabaseAccountsClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	pager := accClient.NewListPager(&cosmosdb.DatabaseAccountsClientListOptions{})
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

			mqlCosmosDbAccount, err := CreateResource(runtime, "azure.subscription.cosmosDbService.account",
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

func fetchDbAccountsByType(ctx context.Context, runtime *plugin.Runtime, conn *connection.AzureConnection, subId string, resourceType string) ([]interface{}, error) {
	resClient, err := armresources.NewClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	filter := fmt.Sprintf("resourceType eq '%s'", resourceType)
	pager := resClient.NewListPager(&armresources.ClientListOptions{
		Filter: &filter,
	})
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

			mqlResource, err := CreateResource(runtime, "azure.subscription.cosmosDbService.account",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(account.ID),
					"name":       llx.StringDataPtr(account.Name),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
					"location":   llx.StringDataPtr(account.Location),
					"kind":       llx.StringDataPtr(account.Kind),
					"type":       llx.StringDataPtr(account.Type),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

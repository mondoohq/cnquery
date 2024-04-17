// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	postgresql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
)

func (a *mqlAzureSubscriptionPostgreSqlService) id() (string, error) {
	return "azure.subscription.postgresql/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionPostgreSqlService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionPostgreSqlService) servers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	dbClient, err := postgresql.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&postgresql.ServersClientListOptions{})
	res := []interface{}{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties := make(map[string](interface{}))

			data, err := json.Marshal(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal([]byte(data), &properties)
			if err != nil {
				return nil, err
			}

			mqlAzurePostgresServer, err := CreateResource(a.MqlRuntime, "azure.subscription.postgreSqlService.server",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(dbServer.ID),
					"name":       llx.StringDataPtr(dbServer.Name),
					"location":   llx.StringDataPtr(dbServer.Location),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":       llx.StringDataPtr(dbServer.Type),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzurePostgresServer)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) databases() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := postgresql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, &json.MarshalerError{}
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &postgresql.DatabasesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.postgreSqlService.database",
				map[string]*llx.RawData{
					"id":        llx.StringDataPtr(entry.ID),
					"name":      llx.StringDataPtr(entry.Name),
					"type":      llx.StringDataPtr(entry.Type),
					"charset":   llx.StringDataPtr(entry.Properties.Charset),
					"collation": llx.StringDataPtr(entry.Properties.Collation),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) firewallRules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}
	dbFirewallClient, err := postgresql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &postgresql.FirewallRulesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlFireWallRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.firewallrule",
				map[string]*llx.RawData{
					"id":             llx.StringDataPtr(entry.ID),
					"name":           llx.StringDataPtr(entry.Name),
					"type":           llx.StringDataPtr(entry.Type),
					"startIpAddress": llx.StringDataPtr(entry.Properties.StartIPAddress),
					"endIpAddress":   llx.StringDataPtr(entry.Properties.EndIPAddress),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFireWallRule)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) configuration() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}
	dbConfClient, err := postgresql.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := dbConfClient.NewListByServerPager(resourceID.ResourceGroup, server, &postgresql.ConfigurationsClientListByServerOptions{})
	res := []interface{}{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureConfiguration, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.configuration",
				map[string]*llx.RawData{
					"id":            llx.StringDataPtr(entry.ID),
					"name":          llx.StringDataPtr(entry.Name),
					"type":          llx.StringDataPtr(entry.Type),
					"value":         llx.StringDataPtr(entry.Properties.Value),
					"description":   llx.StringDataPtr(entry.Properties.Description),
					"defaultValue":  llx.StringDataPtr(entry.Properties.DefaultValue),
					"dataType":      llx.StringDataPtr(entry.Properties.DataType),
					"allowedValues": llx.StringDataPtr(entry.Properties.AllowedValues),
					"source":        llx.StringDataPtr(entry.Properties.Source),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}

	return res, nil
}

func initAzureSubscriptionPostgreSqlServiceServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure postgresql server")
	}
	conn := runtime.Connection.(*connection.AzureConnection)
	res, err := NewResource(runtime, "azure.subscription.postgreSqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	postgreSql := res.(*mqlAzureSubscriptionPostgreSqlService)
	servers := postgreSql.GetServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionPostgreSqlServiceServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure postgresql server does not exist")
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mariadb "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb"
)

func (a *mqlAzureSubscriptionMariaDbService) id() (string, error) {
	return "azure.subscription.mariadb/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionMariaDb(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionMariaDbServiceServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMariaDbServiceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMariaDbService) servers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	dbClient, err := mariadb.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&mariadb.ServersClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties, err := convert.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mariaDbService.server",
				map[string]*llx.RawData{
					"id":         llx.StringData(convert.ToString(dbServer.ID)),
					"name":       llx.StringData(convert.ToString(dbServer.Name)),
					"location":   llx.StringData(convert.ToString(dbServer.Location)),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":       llx.StringData(convert.ToString(dbServer.Type)),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDbServer)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMariaDbServiceServer) databases() ([]interface{}, error) {
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
	dbDatabaseClient, err := mariadb.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &mariadb.DatabasesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mariaDbService.database",
				map[string]*llx.RawData{
					"id":        llx.StringData(convert.ToString(entry.ID)),
					"name":      llx.StringData(convert.ToString(entry.Name)),
					"type":      llx.StringData(convert.ToString(entry.Type)),
					"charset":   llx.StringData(convert.ToString(entry.Properties.Charset)),
					"collation": llx.StringData(convert.ToString(entry.Properties.Collation)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionMariaDbServiceServer) firewallRules() ([]interface{}, error) {
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

	dbFirewallClient, err := mariadb.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &mariadb.FirewallRulesClientListByServerOptions{})
	res := []interface{}{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlFireWallRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.firewallrule",
				map[string]*llx.RawData{
					"id":             llx.StringData(convert.ToString(entry.ID)),
					"name":           llx.StringData(convert.ToString(entry.Name)),
					"type":           llx.StringData(convert.ToString(entry.Type)),
					"startIpAddress": llx.StringData(convert.ToString(entry.Properties.StartIPAddress)),
					"endIpAddress":   llx.StringData(convert.ToString(entry.Properties.EndIPAddress)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFireWallRule)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionMariaDbServiceServer) configuration() ([]interface{}, error) {
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

	dbConfClient, err := mariadb.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbConfClient.NewListByServerPager(resourceID.ResourceGroup, server, &mariadb.ConfigurationsClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureConfiguration, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.configuration",
				map[string]*llx.RawData{
					"id":            llx.StringData(convert.ToString(entry.ID)),
					"name":          llx.StringData(convert.ToString(entry.Name)),
					"type":          llx.StringData(convert.ToString(entry.Type)),
					"value":         llx.StringData(convert.ToString(entry.Properties.Value)),
					"description":   llx.StringData(convert.ToString(entry.Properties.Description)),
					"defaultValue":  llx.StringData(convert.ToString(entry.Properties.DefaultValue)),
					"dataType":      llx.StringData(convert.ToString(entry.Properties.DataType)),
					"allowedValues": llx.StringData(convert.ToString(entry.Properties.AllowedValues)),
					"source":        llx.StringData(convert.ToString(entry.Properties.Source)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

func initAzureSubscriptionMariaDbServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure mariadb server")
	}
	conn := runtime.Connection.(*connection.AzureConnection)
	res, err := NewResource(runtime, "azure.subscription.mariaDbService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	mariadb := res.(*mqlAzureSubscriptionMariaDbService)
	servers := mariadb.GetServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionMariaDbServiceServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure mariadb server does not exist")
}

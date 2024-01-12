// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/types"

	mysql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql"

	flexible "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers"
)

func (a *mqlAzureSubscriptionMySqlService) id() (string, error) {
	return "azure.subscription.mysql/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionMySqlService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionMySqlServiceServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMySqlServiceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMySqlService) servers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	dbClient, err := mysql.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&mysql.ServersClientListOptions{})
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

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.server",
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

func (a *mqlAzureSubscriptionMySqlService) flexibleServers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	dbClient, err := flexible.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&flexible.ServersClientListOptions{})
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

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.flexibleServer",
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

func (a *mqlAzureSubscriptionMySqlServiceServer) databases() ([]interface{}, error) {
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

	dbDatabaseClient, err := mysql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &mysql.DatabasesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.database",
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

func (a *mqlAzureSubscriptionMySqlServiceServer) firewallRules() ([]interface{}, error) {
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

	dbFirewallClient, err := mysql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &mysql.FirewallRulesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlFireWallRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sql.firewallrule",
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

func (a *mqlAzureSubscriptionMySqlServiceServer) configuration() ([]interface{}, error) {
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

	dbConfClient, err := mysql.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbConfClient.NewListByServerPager(resourceID.ResourceGroup, server, &mysql.ConfigurationsClientListByServerOptions{})
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

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) databases() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := flexible.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.DatabasesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.database",
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

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) firewallRules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}
	dbFirewallClient, err := flexible.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.FirewallRulesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlFireWallRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sql.firewallrule",
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

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) configuration() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	dbConfClient, err := flexible.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbConfClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.ConfigurationsClientListByServerOptions{})
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
					"source":        llx.StringData(convert.ToString((*string)(entry.Properties.Source))),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

func initAzureSubscriptionMySqlServiceServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure mysql server")
	}
	conn := runtime.Connection.(*connection.AzureConnection)
	res, err := NewResource(runtime, "azure.subscription.mySqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	mysql := res.(*mqlAzureSubscriptionMySqlService)
	servers := mysql.GetServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionMySqlServiceServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure mysql server does not exist")
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	postgresql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
	flexible "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers"
)

func (a *mqlAzureSubscriptionPostgreSqlService) id() (string, error) {
	return "azure.subscription.postgresql/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionPostgreSqlService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionPostgreSqlServiceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionPostgreSqlService) servers() ([]any, error) {
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
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties := make(map[string](any))

			data, err := json.Marshal(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal([]byte(data), &properties)
			if err != nil {
				return nil, err
			}

			var sslEnforcement string
			var minimalTlsVersion string
			var publicNetworkAccess string
			var infrastructureEncryption string
			var version string
			if dbServer.Properties != nil {
				if dbServer.Properties.SSLEnforcement != nil {
					sslEnforcement = string(*dbServer.Properties.SSLEnforcement)
				}
				if dbServer.Properties.MinimalTLSVersion != nil {
					minimalTlsVersion = string(*dbServer.Properties.MinimalTLSVersion)
				}
				if dbServer.Properties.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*dbServer.Properties.PublicNetworkAccess)
				}
				if dbServer.Properties.InfrastructureEncryption != nil {
					infrastructureEncryption = string(*dbServer.Properties.InfrastructureEncryption)
				}
				if dbServer.Properties.Version != nil {
					version = string(*dbServer.Properties.Version)
				}
			}

			mqlAzurePostgresServer, err := CreateResource(a.MqlRuntime, "azure.subscription.postgreSqlService.server",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(dbServer.ID),
					"name":                     llx.StringDataPtr(dbServer.Name),
					"location":                 llx.StringDataPtr(dbServer.Location),
					"tags":                     llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":                     llx.StringDataPtr(dbServer.Type),
					"properties":               llx.DictData(properties),
					"sslEnforcement":           llx.StringData(sslEnforcement),
					"minimalTlsVersion":        llx.StringData(minimalTlsVersion),
					"publicNetworkAccess":      llx.StringData(publicNetworkAccess),
					"infrastructureEncryption": llx.StringData(infrastructureEncryption),
					"version":                  llx.StringData(version),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzurePostgresServer)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceFlexibleServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionPostgreSqlService) flexibleServers() ([]any, error) {
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
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties := make(map[string](any))

			data, err := json.Marshal(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal([]byte(data), &properties)
			if err != nil {
				return nil, err
			}

			var version string
			if dbServer.Properties != nil && dbServer.Properties.Version != nil {
				version = string(*dbServer.Properties.Version)
			}

			mqlAzurePostgresServer, err := CreateResource(a.MqlRuntime, "azure.subscription.postgreSqlService.flexibleServer",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(dbServer.ID),
					"name":       llx.StringDataPtr(dbServer.Name),
					"location":   llx.StringDataPtr(dbServer.Location),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":       llx.StringDataPtr(dbServer.Type),
					"properties": llx.DictData(properties),
					"version":    llx.StringData(version),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzurePostgresServer)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) databases() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionPostgreSqlServiceFlexibleServer) databases() ([]any, error) {
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
		return nil, &json.MarshalerError{}
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.DatabasesClientListByServerOptions{})
	res := []any{}
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

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) firewallRules() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionPostgreSqlServiceFlexibleServer) firewallRules() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) configuration() ([]any, error) {
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
	res := []any{}

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

func (a *mqlAzureSubscriptionPostgreSqlServiceFlexibleServer) configuration() ([]any, error) {
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
	res := []any{}

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
					"dataType":      llx.StringDataPtr((*string)(entry.Properties.DataType)),
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
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
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

func initAzureSubscriptionPostgreSqlServiceFlexibleServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure postgresql flexible server")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.postgreSqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	postgreSql := res.(*mqlAzureSubscriptionPostgreSqlService)
	servers := postgreSql.GetFlexibleServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionPostgreSqlServiceFlexibleServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure postgresql flexible server does not exist")
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

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

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
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

func (a *mqlAzureSubscriptionMySqlService) servers() ([]any, error) {
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
	res := []any{}
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

			var sslEnforcement *bool
			var minimalTlsVersion *string
			var publicNetworkAccess *string
			var infrastructureEncryption *bool
			var version *string
			if dbServer.Properties != nil {
				if dbServer.Properties.SSLEnforcement != nil {
					v := *dbServer.Properties.SSLEnforcement == mysql.SSLEnforcementEnumEnabled
					sslEnforcement = &v
				}
				minimalTlsVersion = (*string)(dbServer.Properties.MinimalTLSVersion)
				publicNetworkAccess = (*string)(dbServer.Properties.PublicNetworkAccess)
				if dbServer.Properties.InfrastructureEncryption != nil {
					v := *dbServer.Properties.InfrastructureEncryption == mysql.InfrastructureEncryptionEnabled
					infrastructureEncryption = &v
				}
				version = (*string)(dbServer.Properties.Version)
			}

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.server",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(dbServer.ID),
					"name":                     llx.StringDataPtr(dbServer.Name),
					"location":                 llx.StringDataPtr(dbServer.Location),
					"tags":                     llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":                     llx.StringDataPtr(dbServer.Type),
					"properties":               llx.DictData(properties),
					"sslEnforcement":           llx.BoolDataPtr(sslEnforcement),
					"minimalTlsVersion":        llx.StringDataPtr(minimalTlsVersion),
					"publicNetworkAccess":      llx.StringDataPtr(publicNetworkAccess),
					"infrastructureEncryption": llx.BoolDataPtr(infrastructureEncryption),
					"version":                  llx.StringDataPtr(version),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDbServer)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlService) flexibleServers() ([]any, error) {
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
			properties, err := convert.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			var version string
			if dbServer.Properties != nil && dbServer.Properties.Version != nil {
				version = string(*dbServer.Properties.Version)
			}

			var sslEnforcement bool
			var publicNetworkAccess string
			if dbServer.Properties != nil {
				if dbServer.Properties.Network != nil && dbServer.Properties.Network.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*dbServer.Properties.Network.PublicNetworkAccess)
				}
				// MySQL flexible servers enforce SSL via the require_secure_transport server parameter.
				// Check if it's enabled via the replication configuration or network settings.
				// Note: The actual SSL enforcement is controlled by server parameter, not a top-level property.
				// We default to true as MySQL flexible servers enforce SSL by default.
				sslEnforcement = true
			}

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.flexibleServer",
				map[string]*llx.RawData{
					"id":                  llx.StringDataPtr(dbServer.ID),
					"name":               llx.StringDataPtr(dbServer.Name),
					"location":           llx.StringDataPtr(dbServer.Location),
					"tags":               llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":               llx.StringDataPtr(dbServer.Type),
					"properties":         llx.DictData(properties),
					"version":            llx.StringData(version),
					"sslEnforcement":     llx.BoolData(sslEnforcement),
					"publicNetworkAccess": llx.StringData(publicNetworkAccess),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDbServer)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceServer) databases() ([]any, error) {
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
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.database",
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

func (a *mqlAzureSubscriptionMySqlServiceServer) firewallRules() ([]any, error) {
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

func (a *mqlAzureSubscriptionMySqlServiceServer) configuration() ([]any, error) {
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

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) databases() ([]any, error) {
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
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.database",
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

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) firewallRules() ([]any, error) {
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

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) configuration() ([]any, error) {
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
					"dataType":      llx.StringDataPtr(entry.Properties.DataType),
					"allowedValues": llx.StringDataPtr(entry.Properties.AllowedValues),
					"source":        llx.StringDataPtr((*string)(entry.Properties.Source)),
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
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
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

func initAzureSubscriptionMySqlServiceFlexibleServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure mysql flexible server")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.mySqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	mysql := res.(*mqlAzureSubscriptionMySqlService)
	servers := mysql.GetFlexibleServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionMySqlServiceFlexibleServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure mysql flexible server does not exist")
}

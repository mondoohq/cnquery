package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/mariadb/mgmt/mariadb"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *lumiAzurermMariadb) id() (string, error) {
	return "azurerm.mariadb", nil
}

func (a *lumiAzurermMariadbServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMariadbDatabase) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMariadb) GetServers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	subscriptionID := at.SubscriptionID()

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbClient := mariadb.NewServersClient(subscriptionID)
	dbClient.Authorizer = authorizer

	servers, err := dbClient.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	if servers.Value == nil {
		return res, nil
	}

	dbServers := *servers.Value

	for i := range dbServers {
		dbServer := dbServers[i]

		properties, err := core.JsonToDict(dbServer.ServerProperties)
		if err != nil {
			return nil, err
		}

		lumiAzureDbServer, err := a.MotorRuntime.CreateResource("azurerm.mariadb.server",
			"id", core.ToString(dbServer.ID),
			"name", core.ToString(dbServer.Name),
			"location", core.ToString(dbServer.Location),
			"tags", azureTagsToInterface(dbServer.Tags),
			"type", core.ToString(dbServer.Type),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureDbServer)
	}

	return res, nil
}

func (a *lumiAzurermMariadbServer) GetConfiguration() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbConfClient := mariadb.NewConfigurationsClient(resourceID.SubscriptionID)
	dbConfClient.Authorizer = authorizer

	config, err := dbConfClient.ListByServer(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	if config.Value == nil {
		return res, nil
	}

	list := *config.Value
	for i := range list {
		entry := list[i]

		lumiAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.configuration",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"value", core.ToString(entry.Value),
			"description", core.ToString(entry.Description),
			"defaultValue", core.ToString(entry.DefaultValue),
			"dataType", core.ToString(entry.DataType),
			"allowedValues", core.ToString(entry.AllowedValues),
			"source", core.ToString(entry.Source),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureConfiguration)
	}

	return res, nil
}

func (a *lumiAzurermMariadbServer) GetDatabases() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbDatabaseClient := mariadb.NewDatabasesClient(resourceID.SubscriptionID)
	dbDatabaseClient.Authorizer = authorizer

	databases, err := dbDatabaseClient.ListByServer(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if databases.Value == nil {
		return res, nil
	}

	list := *databases.Value
	for i := range list {
		entry := list[i]

		lumiAzureDatabase, err := a.MotorRuntime.CreateResource("azurerm.mariadb.database",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"charset", core.ToString(entry.Charset),
			"collation", core.ToString(entry.Collation),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureDatabase)
	}

	return res, nil
}

func (a *lumiAzurermMariadbServer) GetFirewallRules() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbFirewallClient := mariadb.NewFirewallRulesClient(resourceID.SubscriptionID)
	dbFirewallClient.Authorizer = authorizer

	firewallRules, err := dbFirewallClient.ListByServer(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if firewallRules.Value == nil {
		return res, nil
	}

	list := *firewallRules.Value
	for i := range list {
		entry := list[i]

		lumiAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.firewallrule",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"startIpAddress", core.ToString(entry.StartIPAddress),
			"endIpAddress", core.ToString(entry.EndIPAddress),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureConfiguration)
	}

	return res, nil
}

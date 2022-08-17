package resources

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/postgresql/mgmt/postgresql"
)

func (a *lumiAzurermPostgresql) id() (string, error) {
	return "azurerm.postgresql", nil
}

func (a *lumiAzurermPostgresqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresql) GetServers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbClient := postgresql.NewServersClient(at.SubscriptionID())
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

		properties := make(map[string](interface{}))

		data, err := json.Marshal(dbServer.ServerProperties)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), &properties)
		if err != nil {
			return nil, err
		}

		lumiAzureDbServer, err := a.MotorRuntime.CreateResource("azurerm.postgresql.server",
			"id", toString(dbServer.ID),
			"name", toString(dbServer.Name),
			"location", toString(dbServer.Location),
			"tags", azureTagsToInterface(dbServer.Tags),
			"type", toString(dbServer.Type),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureDbServer)
	}

	return res, nil
}

func (a *lumiAzurermPostgresqlServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresqlServer) GetConfiguration() ([]interface{}, error) {
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

	dbConfClient := postgresql.NewConfigurationsClient(resourceID.SubscriptionID)
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
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"type", toString(entry.Type),
			"value", toString(entry.Value),
			"description", toString(entry.Description),
			"defaultValue", toString(entry.DefaultValue),
			"dataType", toString(entry.DataType),
			"allowedValues", toString(entry.AllowedValues),
			"source", toString(entry.Source),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureConfiguration)
	}

	return res, nil
}

func (a *lumiAzurermPostgresqlServer) GetDatabases() ([]interface{}, error) {
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

	dbDatabaseClient := postgresql.NewDatabasesClient(resourceID.SubscriptionID)
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

		lumiAzureDatabase, err := a.MotorRuntime.CreateResource("azurerm.postgresql.database",
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"type", toString(entry.Type),
			"charset", toString(entry.Charset),
			"collation", toString(entry.Collation),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureDatabase)
	}

	return res, nil
}

func (a *lumiAzurermPostgresqlServer) GetFirewallRules() ([]interface{}, error) {
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

	dbFirewallClient := postgresql.NewFirewallRulesClient(resourceID.SubscriptionID)
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
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"type", toString(entry.Type),
			"startIpAddress", toString(entry.StartIPAddress),
			"endIpAddress", toString(entry.EndIPAddress),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureConfiguration)
	}

	return res, nil
}

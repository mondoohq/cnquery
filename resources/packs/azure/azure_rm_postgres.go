package azure

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/postgresql/mgmt/postgresql"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermPostgresql) id() (string, error) {
	return "azurerm.postgresql", nil
}

func (a *mqlAzurermPostgresqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermPostgresql) GetServers() ([]interface{}, error) {
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

		mqlAzureDbServer, err := a.MotorRuntime.CreateResource("azurerm.postgresql.server",
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
		res = append(res, mqlAzureDbServer)
	}

	return res, nil
}

func (a *mqlAzurermPostgresqlServer) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermPostgresqlServer) GetConfiguration() ([]interface{}, error) {
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

		mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.configuration",
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
		res = append(res, mqlAzureConfiguration)
	}

	return res, nil
}

func (a *mqlAzurermPostgresqlServer) GetDatabases() ([]interface{}, error) {
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

		mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azurerm.postgresql.database",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"charset", core.ToString(entry.Charset),
			"collation", core.ToString(entry.Collation),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureDatabase)
	}

	return res, nil
}

func (a *mqlAzurermPostgresqlServer) GetFirewallRules() ([]interface{}, error) {
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

		mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.firewallrule",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"startIpAddress", core.ToString(entry.StartIPAddress),
			"endIpAddress", core.ToString(entry.EndIPAddress),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureConfiguration)
	}

	return res, nil
}

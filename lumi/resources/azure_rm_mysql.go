package resources

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/mysql/mgmt/mysql"
)

func (a *lumiAzurermMysqlServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMysqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurerm) GetMysqlServers() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	subscriptionID := at.SubscriptionID()

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbClient := mysql.NewServersClient(subscriptionID)
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

		lumiAzureDbServer, err := a.Runtime.CreateResource("azurerm.mysql.server",
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

func (a *lumiAzurermMysqlServer) GetConfiguration() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
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

	dbConfClient := mysql.NewConfigurationsClient(resourceID.SubscriptionID)
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

		lumiAzureConfiguration, err := a.Runtime.CreateResource("azurerm.sql.configuration",
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
func (a *lumiAzurermMysqlServer) GetDatabases() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
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

	dbDatabaseClient := mysql.NewDatabasesClient(resourceID.SubscriptionID)
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

		lumiAzureDatabase, err := a.Runtime.CreateResource("azurerm.mysql.database",
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

func (a *lumiAzurermMysqlServer) GetFirewallRules() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
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

	dbFirewallClient := mysql.NewFirewallRulesClient(resourceID.SubscriptionID)
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

		lumiAzureConfiguration, err := a.Runtime.CreateResource("azurerm.sql.firewallrule",
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

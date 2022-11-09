package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	mariadb "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermMariadb) id() (string, error) {
	return "azurerm.mariadb", nil
}

func (a *mqlAzurermMariadbServer) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermMariadbDatabase) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermMariadb) GetServers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbClient, err := mariadb.NewServersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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
			properties, err := core.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureDbServer, err := a.MotorRuntime.CreateResource("azurerm.mariadb.server",
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
	}
	return res, nil
}

func (a *mqlAzurermMariadbServer) GetConfiguration() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
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
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbConfClient, err := mariadb.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.configuration",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"value", core.ToString(entry.Properties.Value),
				"description", core.ToString(entry.Properties.Description),
				"defaultValue", core.ToString(entry.Properties.DefaultValue),
				"dataType", core.ToString(entry.Properties.DataType),
				"allowedValues", core.ToString(entry.Properties.AllowedValues),
				"source", core.ToString(entry.Properties.Source),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

func (a *mqlAzurermMariadbServer) GetDatabases() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
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
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := mariadb.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azurerm.mariadb.database",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"charset", core.ToString(entry.Properties.Charset),
				"collation", core.ToString(entry.Properties.Collation),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}

	return res, nil
}

func (a *mqlAzurermMariadbServer) GetFirewallRules() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
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
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbFirewallClient, err := mariadb.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.firewallrule",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"startIpAddress", core.ToString(entry.Properties.StartIPAddress),
				"endIpAddress", core.ToString(entry.Properties.EndIPAddress),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}

	return res, nil
}

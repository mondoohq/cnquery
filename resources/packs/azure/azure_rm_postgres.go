package azure

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	postgresql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
	azure "go.mondoo.com/cnquery/motor/providers/microsoft/azure"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurePostgresql) id() (string, error) {
	return "azure.postgresql", nil
}

func (a *mqlAzurePostgresqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurePostgresql) GetServers() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbClient, err := postgresql.NewServersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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

			mqlAzureDbServer, err := a.MotorRuntime.CreateResource("azure.postgresql.server",
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

func (a *mqlAzurePostgresqlServer) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurePostgresqlServer) GetConfiguration() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := azure.ParseResourceID(id)
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

	dbConfClient, err := postgresql.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azure.sql.configuration",
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

func (a *mqlAzurePostgresqlServer) GetDatabases() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := azure.ParseResourceID(id)
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

	dbDatabaseClient, err := postgresql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azure.postgresql.database",
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

func (a *mqlAzurePostgresqlServer) GetFirewallRules() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := azure.ParseResourceID(id)
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

	dbFirewallClient, err := postgresql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azure.sql.firewallrule",
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

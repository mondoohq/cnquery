package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mysql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql"
	flexible "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers"
	azure "go.mondoo.com/cnquery/motor/providers/microsoft/azure"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscriptionMysqlService) init(args *resources.Args) (*resources.Args, AzureSubscriptionMysqlService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	(*args)["subscriptionId"] = at.SubscriptionID()

	return args, nil, nil
}

func (a *mqlAzureSubscriptionMysqlService) id() (string, error) {
	subId, err := a.SubscriptionId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/subscriptions/%s/mySqlService", subId), nil
}

func (a *mqlAzureSubscriptionMysqlServiceServer) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionMysqlServiceFlexibleServer) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionMysqlServiceDatabase) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionMysqlService) GetServers() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbClient, err := mysql.NewServersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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
			properties, err := core.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureDbServer, err := a.MotorRuntime.CreateResource("azure.subscription.mysqlService.server",
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

func (a *mqlAzureSubscriptionMysqlService) GetFlexibleServers() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbClient, err := flexible.NewServersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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
			properties, err := core.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureDbServer, err := a.MotorRuntime.CreateResource("azure.subscription.mysqlService.flexibleServer",
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

func (a *mqlAzureSubscriptionMysqlServiceServer) GetConfiguration() ([]interface{}, error) {
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

	dbConfClient, err := mysql.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azure.subscription.sqlService.configuration",
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

func (a *mqlAzureSubscriptionMysqlServiceServer) GetDatabases() ([]interface{}, error) {
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

	dbDatabaseClient, err := mysql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azure.subscription.mysqlService.database",
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

func (a *mqlAzureSubscriptionMysqlServiceFlexibleServer) GetDatabases() ([]interface{}, error) {
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

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := flexible.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azure.subscription.mysqlService.database",
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

func (a *mqlAzureSubscriptionMysqlServiceServer) GetFirewallRules() ([]interface{}, error) {
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

	dbFirewallClient, err := mysql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azure.subscription.sqlService.firewallrule",
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

func (a *mqlAzureSubscriptionMysqlServiceFlexibleServer) GetConfiguration() ([]interface{}, error) {
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

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbConfClient, err := flexible.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azure.subscription.sqlService.configuration",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"value", core.ToString(entry.Properties.Value),
				"description", core.ToString(entry.Properties.Description),
				"defaultValue", core.ToString(entry.Properties.DefaultValue),
				"dataType", core.ToString(entry.Properties.DataType),
				"allowedValues", core.ToString(entry.Properties.AllowedValues),
				"source", core.ToString((*string)(entry.Properties.Source)),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMysqlServiceFlexibleServer) GetFirewallRules() ([]interface{}, error) {
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

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbFirewallClient, err := flexible.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
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
			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azure.subscription.sqlService.firewallrule",
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

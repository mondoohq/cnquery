package resources

import (
	"context"
	"errors"

	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/postgresql/mgmt/postgresql"
)

func (a *lumiAzurerm) id() (string, error) {
	return "azurerm", nil
}

func azureTagsToInterface(data map[string]*string) map[string]interface{} {
	labels := make(map[string]interface{})
	for key := range data {
		labels[key] = toString(data[key])
	}
	return labels
}

func (a *lumiAzurerm) GetVms() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	subscriptionID := at.SubscriptionID()

	// list compute instances
	// TODO: iterate over all resource groups
	resourceGroup := "demo"
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer

	virtualMachines, err := vmClient.List(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range virtualMachines.Values() {
		vm := virtualMachines.Values()[i]

		properties := make(map[string](interface{}))

		data, err := json.Marshal(vm.VirtualMachineProperties)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), &properties)
		if err != nil {
			return nil, err
		}

		lumiAzureVm, err := a.Runtime.CreateResource("azurerm.compute.vm",
			"id", toString(vm.ID),
			"name", toString(vm.Name),
			"location", toString(vm.Location),
			"tags", azureTagsToInterface(vm.Tags),
			"type", toString(vm.Type),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureVm)
	}

	return res, nil
}

func (a *lumiAzurerm) GetSqlServers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzurerm) GetPostgresqlServers() ([]interface{}, error) {
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

	dbClient := postgresql.NewServersClient(subscriptionID)
	dbClient.Authorizer = authorizer

	servers, err := dbClient.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	if servers.Value == nil {
		return res, nil
	}

	pgServers := *servers.Value

	for i := range pgServers {
		pgServer := pgServers[i]

		properties := make(map[string](interface{}))

		data, err := json.Marshal(pgServer.ServerProperties)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(data), &properties)
		if err != nil {
			return nil, err
		}

		lumiAzureVm, err := a.Runtime.CreateResource("azurerm.postgresql.server",
			"id", toString(pgServer.ID),
			"name", toString(pgServer.Name),
			"location", toString(pgServer.Location),
			"tags", azureTagsToInterface(pgServer.Tags),
			"type", toString(pgServer.Type),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureVm)
	}

	return res, nil
}

func (a *lumiAzurermPostgresqlServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresqlServer) GetConfiguration() ([]interface{}, error) {
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

		lumiAzureConfiguration, err := a.Runtime.CreateResource("azurerm.configuration",
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

		lumiAzurePgDatabase, err := a.Runtime.CreateResource("azurerm.postgresql.database",
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"type", toString(entry.Type),
			"charset", toString(entry.Charset),
			"collation", toString(entry.Collation),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzurePgDatabase)
	}

	return res, nil
}

func (a *lumiAzurermPostgresqlServer) GetFirewallRules() ([]interface{}, error) {

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

		lumiAzureConfiguration, err := a.Runtime.CreateResource("azurerm.postgresql.firewallrule",
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

func (a *lumiAzurermConfiguration) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresqlFirewallrule) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermComputeVm) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermStorageAccount) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermStorageBlob) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMssqlServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMssqlDatabase) id() (string, error) {
	return a.Id()
}

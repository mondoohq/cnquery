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

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	subscriptionID := at.SubscriptionID()

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

func (a *lumiAzurermComputeVm) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermStorageAccount) id() (string, error) {
	return a.Id()
}

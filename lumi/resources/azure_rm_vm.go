package resources

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
)

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

func (a *lumiAzurermComputeVm) id() (string, error) {
	return a.Id()
}

package resources

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"go.mondoo.io/mondoo/lumi"
)

func (a *lumiAzurermCompute) id() (string, error) {
	return "azurerm.compute", nil
}

func (a *lumiAzurermCompute) GetDisks() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := compute.NewDisksClient(at.SubscriptionID())
	client.Authorizer = authorizer

	disks, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range disks.Values() {
		disk := disks.Values()[i]

		lumiAzureDisk, err := diskToLumi(a.Runtime, disk)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureDisk)
	}

	return res, nil
}

func diskToLumi(runtime *lumi.Runtime, disk compute.Disk) (lumi.ResourceType, error) {
	properties, err := jsonToDict(disk.DiskProperties)
	if err != nil {
		return nil, err
	}

	sku, err := jsonToDict(disk.Sku)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("azurerm.compute.disk",
		"id", toString(disk.ID),
		"name", toString(disk.Name),
		"location", toString(disk.Location),
		"tags", azureTagsToInterface(disk.Tags),
		"type", toString(disk.Type),
		"managedBy", toString(disk.ManagedBy),
		"managedByExtended", toStringSlice(disk.ManagedByExtended),
		"zones", toStringSlice(disk.Zones),
		"sku", sku,
		"properties", properties,
	)
}

func (a *lumiAzurermComputeDisk) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermCompute) GetVms() ([]interface{}, error) {
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

		properties, err := jsonToDict(vm.VirtualMachineProperties)
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

func (a *lumiAzurermComputeVm) GetExtensions() ([]interface{}, error) {
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

	vm, err := resourceID.Component("virtualMachines")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := compute.NewVirtualMachineExtensionsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	extensions, err := client.List(ctx, resourceID.ResourceGroup, vm, "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if extensions.Value == nil {
		return res, nil
	}

	list := *extensions.Value

	for i := range list {
		entry := list[i]

		dict, err := jsonToDict(entry.VirtualMachineExtensionProperties)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *lumiAzurermComputeVm) GetOsDisk() (interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	propertiesDict, err := a.Properties()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(propertiesDict)
	if err != nil {
		return nil, err
	}

	var properties compute.VirtualMachineProperties
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	if properties.StorageProfile == nil || properties.StorageProfile.OsDisk == nil || properties.StorageProfile.OsDisk.ManagedDisk == nil || properties.StorageProfile.OsDisk.ManagedDisk.ID == nil {
		return nil, errors.New("could not determine os disk from vm storage profile")
	}

	resourceID, err := at.ParseResourceID(*properties.StorageProfile.OsDisk.ManagedDisk.ID)
	if err != nil {
		return nil, err
	}

	diskName, err := resourceID.Component("disks")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := compute.NewDisksClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName)
	if err != nil {
		return nil, err
	}

	return diskToLumi(a.Runtime, disk)
}

func (a *lumiAzurermComputeVm) GetDataDisks() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	propertiesDict, err := a.Properties()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(propertiesDict)
	if err != nil {
		return nil, err
	}

	var properties compute.VirtualMachineProperties
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	if properties.StorageProfile == nil || properties.StorageProfile.DataDisks == nil {
		return nil, errors.New("could not determine os disk from vm storage profile")
	}

	dataDisks := *properties.StorageProfile.DataDisks

	res := []interface{}{}
	for i := range dataDisks {
		dataDisk := dataDisks[i]

		resourceID, err := at.ParseResourceID(*dataDisk.ManagedDisk.ID)
		if err != nil {
			return nil, err
		}

		diskName, err := resourceID.Component("disks")
		if err != nil {
			return nil, err
		}

		ctx := context.Background()
		authorizer, err := at.Authorizer()
		if err != nil {
			return nil, err
		}

		client := compute.NewDisksClient(resourceID.SubscriptionID)
		client.Authorizer = authorizer

		disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName)
		if err != nil {
			return nil, err
		}

		lumiDisk, err := diskToLumi(a.Runtime, disk)
		if err != nil {
			return nil, err
		}

		res = append(res, lumiDisk)
	}

	return res, nil
}

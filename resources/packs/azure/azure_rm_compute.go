package azure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *lumiAzurermCompute) id() (string, error) {
	return "azurerm.compute", nil
}

func (a *lumiAzurermCompute) GetDisks() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

		lumiAzureDisk, err := diskToLumi(a.MotorRuntime, disk)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzureDisk)
	}

	return res, nil
}

func diskToLumi(runtime *lumi.Runtime, disk compute.Disk) (lumi.ResourceType, error) {
	properties, err := core.JsonToDict(disk.DiskProperties)
	if err != nil {
		return nil, err
	}

	sku, err := core.JsonToDict(disk.Sku)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("azurerm.compute.disk",
		"id", core.ToString(disk.ID),
		"name", core.ToString(disk.Name),
		"location", core.ToString(disk.Location),
		"tags", azureTagsToInterface(disk.Tags),
		"type", core.ToString(disk.Type),
		"managedBy", core.ToString(disk.ManagedBy),
		"managedByExtended", core.ToStringSlice(disk.ManagedByExtended),
		"zones", core.ToStringSlice(disk.Zones),
		"sku", sku,
		"properties", properties,
	)
}

func (a *lumiAzurermComputeDisk) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermCompute) GetVms() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer

	virtualMachines, err := vmClient.ListAll(ctx, "", "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range virtualMachines.Values() {
		vm := virtualMachines.Values()[i]

		properties, err := core.JsonToDict(vm.VirtualMachineProperties)
		if err != nil {
			return nil, err
		}

		lumiAzureVm, err := a.MotorRuntime.CreateResource("azurerm.compute.vm",
			"id", core.ToString(vm.ID),
			"name", core.ToString(vm.Name),
			"location", core.ToString(vm.Location),
			"tags", azureTagsToInterface(vm.Tags),
			"type", core.ToString(vm.Type),
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

		dict, err := core.JsonToDict(entry.VirtualMachineExtensionProperties)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *lumiAzurermComputeVm) GetOsDisk() (interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

	return diskToLumi(a.MotorRuntime, disk)
}

func (a *lumiAzurermComputeVm) GetDataDisks() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
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

		lumiDisk, err := diskToLumi(a.MotorRuntime, disk)
		if err != nil {
			return nil, err
		}

		res = append(res, lumiDisk)
	}

	return res, nil
}

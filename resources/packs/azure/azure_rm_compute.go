package azure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermCompute) id() (string, error) {
	return "azurerm.compute", nil
}

func (a *mqlAzurermCompute) GetDisks() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := compute.NewDisksClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&compute.DisksClientListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for pager.More() {
		disks, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, disk := range disks.Value {
			mqlAzureDisk, err := diskToMql(a.MotorRuntime, *disk)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDisk)
		}
	}

	return res, nil
}

func diskToMql(runtime *resources.Runtime, disk compute.Disk) (resources.ResourceType, error) {
	properties, err := core.JsonToDict(disk.Properties)
	if err != nil {
		return nil, err
	}

	sku, err := core.JsonToDict(disk.SKU)
	if err != nil {
		return nil, err
	}

	managedByExtended := []string{}
	for _, mbe := range disk.ManagedByExtended {
		if mbe != nil {
			managedByExtended = append(managedByExtended, *mbe)
		}
	}
	zones := []string{}
	for _, z := range disk.Zones {
		if z != nil {
			zones = append(zones, *z)
		}
	}
	return runtime.CreateResource("azurerm.compute.disk",
		"id", core.ToString(disk.ID),
		"name", core.ToString(disk.Name),
		"location", core.ToString(disk.Location),
		"tags", azureTagsToInterface(disk.Tags),
		"type", core.ToString(disk.Type),
		"managedBy", core.ToString(disk.ManagedBy),
		"managedByExtended", core.ToStringSlice(&managedByExtended),
		"zones", core.ToStringSlice(&zones),
		"sku", sku,
		"properties", properties,
	)
}

func (a *mqlAzurermComputeDisk) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermCompute) GetVms() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	// list compute instances
	vmClient, err := compute.NewVirtualMachinesClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := vmClient.NewListAllPager(&compute.VirtualMachinesClientListAllOptions{})

	res := []interface{}{}
	for pager.More() {
		vms, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vm := range vms.Value {
			properties, err := core.JsonToDict(vm.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureVm, err := a.MotorRuntime.CreateResource("azurerm.compute.vm",
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
			res = append(res, mqlAzureVm)
		}
	}

	return res, nil
}

func (a *mqlAzurermComputeVm) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermComputeVm) GetExtensions() ([]interface{}, error) {
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

	vm, err := resourceID.Component("virtualMachines")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := compute.NewVirtualMachineExtensionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	extensions, err := client.List(ctx, resourceID.ResourceGroup, vm, &compute.VirtualMachineExtensionsClientListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if extensions.Value == nil {
		return res, nil
	}

	list := extensions.Value

	for i := range list {
		entry := list[i]

		dict, err := core.JsonToDict(entry.Properties)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *mqlAzurermComputeVm) GetOsDisk() (interface{}, error) {
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

	if properties.StorageProfile == nil || properties.StorageProfile.OSDisk == nil || properties.StorageProfile.OSDisk.ManagedDisk == nil || properties.StorageProfile.OSDisk.ManagedDisk.ID == nil {
		return nil, errors.New("could not determine os disk from vm storage profile")
	}

	resourceID, err := at.ParseResourceID(*properties.StorageProfile.OSDisk.ManagedDisk.ID)
	if err != nil {
		return nil, err
	}

	diskName, err := resourceID.Component("disks")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName, &compute.DisksClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return diskToMql(a.MotorRuntime, disk.Disk)
}

func (a *mqlAzurermComputeVm) GetDataDisks() ([]interface{}, error) {
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

	dataDisks := properties.StorageProfile.DataDisks

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
		token, err := at.GetTokenCredential()
		if err != nil {
			return nil, err
		}

		client, err := compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
		if err != nil {
			return nil, err
		}
		disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName, &compute.DisksClientGetOptions{})
		if err != nil {
			return nil, err
		}

		mqlDisk, err := diskToMql(a.MotorRuntime, disk.Disk)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlDisk)
	}

	return res, nil
}

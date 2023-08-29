// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/azure/connection"
	"go.mondoo.com/cnquery/types"
)

func (a *mqlAzureSubscriptionCompute) id() (string, error) {
	return "azure.subscription.compute/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionCompute(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCompute) vms() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	// list compute instances
	vmClient, err := compute.NewVirtualMachinesClient(subId, token, &arm.ClientOptions{})
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
			properties, err := convert.JsonToDict(vm.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureVm, err := CreateResource(a.MqlRuntime, "azure.subscription.compute.vm",
				map[string]*llx.RawData{
					"id":         llx.StringData(convert.ToString(vm.ID)),
					"name":       llx.StringData(convert.ToString(vm.Name)),
					"location":   llx.StringData(convert.ToString(vm.Location)),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(vm.Tags), types.String),
					"type":       llx.StringData(convert.ToString(vm.Type)),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureVm)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeVm) extensions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// id is a azure resource id
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	vm, err := resourceID.Component("virtualMachines")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token := conn.Token()
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

		dict, err := convert.JsonToDict(entry.Properties)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *mqlAzureSubscriptionCompute) disks() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := compute.NewDisksClient(subId, token, &arm.ClientOptions{})
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
			mqlAzureDisk, err := diskToMql(a.MqlRuntime, *disk)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDisk)
		}
	}

	return res, nil
}

func diskToMql(runtime *plugin.Runtime, disk compute.Disk) (*mqlAzureSubscriptionComputeDisk, error) {
	properties, err := convert.JsonToDict(disk.Properties)
	if err != nil {
		return nil, err
	}

	sku, err := convert.JsonToDict(disk.SKU)
	if err != nil {
		return nil, err
	}

	managedByExtended := []interface{}{}
	for _, mbe := range disk.ManagedByExtended {
		if mbe != nil {
			managedByExtended = append(managedByExtended, *mbe)
		}
	}
	zones := []interface{}{}
	for _, z := range disk.Zones {
		if z != nil {
			zones = append(zones, *z)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.compute.disk",
		map[string]*llx.RawData{
			"id":                llx.StringData(convert.ToString(disk.ID)),
			"name":              llx.StringData(convert.ToString(disk.Name)),
			"location":          llx.StringData(convert.ToString(disk.Location)),
			"tags":              llx.MapData(convert.PtrMapStrToInterface(disk.Tags), types.String),
			"type":              llx.StringData(convert.ToString(disk.Type)),
			"managedBy":         llx.StringData(convert.ToString(disk.ManagedBy)),
			"managedByExtended": llx.ArrayData(managedByExtended, types.String),
			"zones":             llx.ArrayData(zones, types.String),
			"sku":               llx.DictData(sku),
			"properties":        llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionComputeDisk), nil
}

func (a *mqlAzureSubscriptionComputeVm) osDisk() (*mqlAzureSubscriptionComputeDisk, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	propertiesDict := a.Properties.Data
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

	resourceID, err := ParseResourceID(*properties.StorageProfile.OSDisk.ManagedDisk.ID)
	if err != nil {
		return nil, err
	}

	diskName, err := resourceID.Component("disks")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token := conn.Token()

	client, err := compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName, &compute.DisksClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return diskToMql(a.MqlRuntime, disk.Disk)
}

func (a *mqlAzureSubscriptionComputeVm) dataDisks() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	propertiesDict := a.Properties.Data
	data, err := json.Marshal(propertiesDict)
	if err != nil {
		return nil, err
	}

	token := conn.Token()

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
	for _, dd := range dataDisks {
		resourceID, err := ParseResourceID(*dd.ManagedDisk.ID)
		if err != nil {
			return nil, err
		}

		diskName, err := resourceID.Component("disks")
		if err != nil {
			return nil, err
		}

		ctx := context.Background()
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

		mqlDisk, err := diskToMql(a.MqlRuntime, disk.Disk)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlDisk)
	}

	return res, nil
}

func (a *mqlAzureSubscriptionComputeVm) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeDisk) id() (string, error) {
	return a.Id.Data, nil
}

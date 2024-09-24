// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAzureSubscriptionComputeService) id() (string, error) {
	return "azure.subscription.compute/" + a.SubscriptionId.Data, nil
}

func getState(vm compute.VirtualMachineInstanceView) string {
	if vm.Statuses == nil {
		return "unknown"
	}
	state := "unknown"
	for _, s := range vm.Statuses {
		if s.Code != nil && *s.Code == "PowerState/running" {
			state = "running"
		}
		if s.Code != nil && *s.Code == "PowerState/deallocated" {
			state = "stopped"
		}
	}
	return state
}

func initAzureSubscriptionComputeService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionComputeService) vms() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	// list compute instances
	vmClient, err := compute.NewVirtualMachinesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

			mqlAzureVm, err := CreateResource(a.MqlRuntime, "azure.subscription.computeService.vm",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(vm.ID),
					"name":       llx.StringDataPtr(vm.Name),
					"location":   llx.StringDataPtr(vm.Location),
					"zones":      llx.ArrayData(convert.SliceStrPtrToInterface(vm.Zones), types.String),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(vm.Tags), types.String),
					"type":       llx.StringDataPtr(vm.Type),
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

func (a *mqlAzureSubscriptionComputeServiceVm) state() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// id is a Azure resource ID
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return "", err
	}

	vm, err := resourceID.Component("virtualMachines")
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	token := conn.Token()
	if err != nil {
		return "", err
	}

	client, err := compute.NewVirtualMachinesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return "", err
	}

	view, err := client.InstanceView(ctx, resourceID.ResourceGroup, vm, &compute.VirtualMachinesClientInstanceViewOptions{})
	if err != nil {
		return "", err
	}
	return getState(view.VirtualMachineInstanceView), nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) isRunning() (bool, error) {
	state := a.GetState()
	if state.Error != nil {
		return false, state.Error
	}
	return state.Data == "running", nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) extensions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// id is a Azure resource ID
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

	client, err := compute.NewVirtualMachineExtensionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

func (a *mqlAzureSubscriptionComputeService) disks() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := compute.NewDisksClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

func diskToMql(runtime *plugin.Runtime, disk compute.Disk) (*mqlAzureSubscriptionComputeServiceDisk, error) {
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

	res, err := CreateResource(runtime, "azure.subscription.computeService.disk",
		map[string]*llx.RawData{
			"id":                llx.StringDataPtr(disk.ID),
			"name":              llx.StringDataPtr(disk.Name),
			"location":          llx.StringDataPtr(disk.Location),
			"tags":              llx.MapData(convert.PtrMapStrToInterface(disk.Tags), types.String),
			"type":              llx.StringDataPtr(disk.Type),
			"managedBy":         llx.StringDataPtr(disk.ManagedBy),
			"managedByExtended": llx.ArrayData(managedByExtended, types.String),
			"zones":             llx.ArrayData(zones, types.String),
			"sku":               llx.DictData(sku),
			"properties":        llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionComputeServiceDisk), nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) osDisk() (*mqlAzureSubscriptionComputeServiceDisk, error) {
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

	client, err := compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	disk, err := client.Get(ctx, resourceID.ResourceGroup, diskName, &compute.DisksClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return diskToMql(a.MqlRuntime, disk.Disk)
}

func (a *mqlAzureSubscriptionComputeServiceVm) dataDisks() ([]interface{}, error) {
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
		return nil, errors.New("could not determine data disks from vm storage profile")
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

		client, err := compute.NewDisksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
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

func (a *mqlAzureSubscriptionComputeServiceVm) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceDisk) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) publicIpAddresses() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	token := conn.Token()
	resourceId, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	subId := resourceId.SubscriptionID
	props := a.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	propsDict := (props.Data).(map[string]interface{})
	networkInterface, ok := propsDict["networkProfile"]
	if !ok {
		return nil, errors.New("cannot find network profile on vm, not retrieving ip addresses")
	}
	var networkInterfaces compute.NetworkProfile

	data, err := json.Marshal(networkInterface)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), &networkInterfaces)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}

	ctx := context.Background()
	nicClient, err := network.NewInterfacesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	ipClient, err := network.NewPublicIPAddressesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	for _, iface := range networkInterfaces.NetworkInterfaces {
		resource, err := ParseResourceID(*iface.ID)
		if err != nil {
			return nil, err
		}

		name, err := resource.Component("networkInterfaces")
		if err != nil {
			return nil, err
		}
		networkInterface, err := nicClient.Get(ctx, resource.ResourceGroup, name, &network.InterfacesClientGetOptions{})
		if err != nil {
			return nil, err
		}

		for _, config := range networkInterface.Interface.Properties.IPConfigurations {
			ip := config.Properties.PublicIPAddress
			if ip != nil {
				publicIPID := *ip.ID
				publicIpResource, err := ParseResourceID(publicIPID)
				if err != nil {
					return nil, errors.New("invalid network information for resource " + publicIPID)
				}

				ipAddrName, err := publicIpResource.Component("publicIPAddresses")
				if err != nil {
					return nil, errors.New("invalid network information for resource " + publicIPID)
				}
				ipAddress, err := ipClient.Get(ctx, publicIpResource.ResourceGroup, ipAddrName, &network.PublicIPAddressesClientGetOptions{})
				if err != nil {
					return nil, err
				}
				mqlIpAddress, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.ipAddress",
					map[string]*llx.RawData{
						"id":        llx.StringDataPtr(ipAddress.ID),
						"name":      llx.StringDataPtr(ipAddress.Name),
						"location":  llx.StringDataPtr(ipAddress.Location),
						"tags":      llx.MapData(convert.PtrMapStrToInterface(ipAddress.Tags), types.String),
						"ipAddress": llx.StringDataPtr(ipAddress.Properties.IPAddress),
						"type":      llx.StringDataPtr(ipAddress.Type),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlIpAddress)
			}
		}
	}

	return res, nil
}

func initAzureSubscriptionComputeServiceVm(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure compute vm instance")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.computeService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := res.(*mqlAzureSubscriptionComputeService)
	vms := computeSvc.GetVms()
	if vms.Error != nil {
		return nil, nil, vms.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range vms.Data {
		vm := entry.(*mqlAzureSubscriptionComputeServiceVm)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure compute instance does not exist")
}

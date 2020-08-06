package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"go.mondoo.io/mondoo/lumi"
)

func (a *lumiAzurerm) GetDisks() ([]interface{}, error) {
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

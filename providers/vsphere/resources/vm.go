// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	vmwaretypes "github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/vsphere/connection"
	"go.mondoo.com/cnquery/v11/providers/vsphere/resources/resourceclient"
	"go.mondoo.com/cnquery/v11/types"
)

// extractTagKeys extracts tag keys from vmware Tag slice
func extractTagKeys(tags []vmwaretypes.Tag) []string {
	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = tag.Key
	}
	return tagKeys
}

// stringSliceToInterface converts []string to []interface{}
func stringSliceToInterface(slice []string) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

func newMqlVm(runtime *plugin.Runtime, vm *object.VirtualMachine, vmInfo *mo.VirtualMachine) (*mqlVsphereVm, error) {
	props, err := resourceclient.VmProperties(vmInfo)
	if err != nil {
		return nil, err
	}

	var name string
	if vmInfo != nil && vmInfo.Config != nil {
		name = vmInfo.Config.Name
	}

	// Extract tags from vmInfo.Tag field (simpler than vAPI REST client)
	tags := extractTagKeys(vmInfo.Tag)

	mqlVm, err := CreateResource(runtime, "vsphere.vm", map[string]*llx.RawData{
		"moid":          llx.StringData(vm.Reference().Encode()),
		"name":          llx.StringData(name),
		"properties":    llx.DictData(props),
		"inventoryPath": llx.StringData(vm.InventoryPath),
		"tags":          llx.ArrayData(stringSliceToInterface(tags), types.String),
	})
	if err != nil {
		return nil, err
	}

	mqlVm.(*mqlVsphereVm).vm = vmInfo
	return mqlVm.(*mqlVsphereVm), nil
}

type mqlVsphereVmInternal struct {
	vm *mo.VirtualMachine
}

func (v *mqlVsphereVm) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlVsphereVm) advancedSettings() (map[string]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	vm, err := vClient.VirtualMachineByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return resourceclient.AdvancedSettings(vm)
}

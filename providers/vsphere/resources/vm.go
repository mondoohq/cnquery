// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/vsphere/connection"
	"go.mondoo.com/cnquery/v11/providers/vsphere/resources/resourceclient"
	"go.mondoo.com/cnquery/v11/types"
)

func newMqlVm(runtime *plugin.Runtime, vm *object.VirtualMachine, vmInfo *mo.VirtualMachine) (*mqlVsphereVm, error) {
	props, err := resourceclient.VmProperties(vmInfo)
	if err != nil {
		return nil, err
	}

	var name string
	var tags []string
	if vmInfo != nil && vmInfo.Config != nil {
		name = vmInfo.Config.Name
		// Try both approaches for testing
		simpleTags := extractTagKeys(vmInfo.Tag)

		// Get vAPI tags using the connection
		conn := runtime.Connection.(*connection.VsphereConnection)
		vClient := resourceclient.New(conn.Client())
		ctx := context.Background()

		// Get vAPI tags using the connection config
		vapiTags := GetTags(ctx, vm.Reference(), vClient.Client.Client, conn.Conf)
		// Use vAPI tags if available, fallback to simple tags
		if len(vapiTags) > 0 {
			tags = vapiTags
		} else {
			tags = simpleTags
		}
	}

	mqlVm, err := CreateResource(runtime, "vsphere.vm", map[string]*llx.RawData{
		"moid":          llx.StringData(vm.Reference().Encode()),
		"name":          llx.StringData(name),
		"properties":    llx.DictData(props),
		"inventoryPath": llx.StringData(vm.InventoryPath),
		"tags":          llx.ArrayData(convert.SliceAnyToInterface(tags), types.String),
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

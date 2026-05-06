// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.com/mql/v13/providers/vsphere/connection"
	"go.mondoo.com/mql/v13/providers/vsphere/resources/resourceclient"
)

type mqlVsphereVmInternal struct {
	vm *mo.VirtualMachine
}

func (v *mqlVsphereVm) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlVsphereVm) advancedSettings() (map[string]any, error) {
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

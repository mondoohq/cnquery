// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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

// kmsCluster resolves the typed vsphere.kmsCluster providing this VM's
// encryption key. Walks vsphere.kmsClusters (single SOAP call, cached) and
// matches by clusterId; null when the VM isn't encrypted or its provider isn't
// in the registered list.
func (v *mqlVsphereVm) kmsCluster() (*mqlVsphereKmsCluster, error) {
	if v.vm == nil || v.vm.Config == nil || v.vm.Config.KeyId == nil || v.vm.Config.KeyId.ProviderId == nil {
		v.KmsCluster.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	providerId := v.vm.Config.KeyId.ProviderId.Id
	if providerId == "" {
		v.KmsCluster.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := CreateResource(v.MqlRuntime, "vsphere", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	clusters := res.(*mqlVsphere).GetKmsClusters()
	if clusters.Error != nil {
		return nil, clusters.Error
	}
	for _, c := range clusters.Data {
		cluster := c.(*mqlVsphereKmsCluster)
		if cluster.ClusterId.Data == providerId {
			return cluster, nil
		}
	}
	v.KmsCluster.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

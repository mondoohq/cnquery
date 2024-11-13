// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/object"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/vsphere/connection"
	"go.mondoo.com/cnquery/v11/providers/vsphere/resources/resourceclient"
)

func newVsphereHostResources(vClient *resourceclient.Client, runtime *plugin.Runtime, vhosts []*object.HostSystem) ([]interface{}, error) {
	mqlHosts := make([]interface{}, len(vhosts))
	for i, h := range vhosts {

		hostInfo, err := resourceclient.HostInfo(h)
		if err != nil {
			return nil, err
		}

		props, err := resourceclient.HostProperties(hostInfo)
		if err != nil {
			return nil, err
		}

		var name string
		if hostInfo != nil {
			name = hostInfo.Name
		}

		mqlHost, err := CreateResource(runtime, "vsphere.host", map[string]*llx.RawData{
			"moid":          llx.StringData(h.Reference().Encode()),
			"name":          llx.StringData(name),
			"properties":    llx.DictData(props),
			"inventoryPath": llx.StringData(h.InventoryPath),
		})
		if err != nil {
			return nil, err
		}
		mqlHost.(*mqlVsphereHost).host = hostInfo

		mqlHosts[i] = mqlHost
	}

	return mqlHosts, nil
}

func (v *mqlVsphereDatacenter) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlVsphereDatacenter) hosts() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	dc, err := client.Datacenter(path)
	if err != nil {
		return nil, err
	}

	vhosts, err := client.ListHosts(dc, nil)
	if err != nil {
		return nil, fmt.Errorf("error listing hosts for datacenter %s: %w", dc.InventoryPath, err)
	}
	return newVsphereHostResources(client, v.MqlRuntime, vhosts)
}

func (v *mqlVsphereDatacenter) clusters() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	dc, err := client.Datacenter(path)
	if err != nil {
		return nil, err
	}

	vCluster, err := client.ListClusters(dc)
	if err != nil {
		return nil, err
	}

	mqlClusters := make([]interface{}, len(vCluster))
	for i, c := range vCluster {

		props, err := client.ClusterProperties(c)
		if err != nil {
			return nil, err
		}

		mqlCluster, err := CreateResource(v.MqlRuntime, "vsphere.cluster", map[string]*llx.RawData{
			"moid":          llx.StringData(c.Reference().Encode()),
			"name":          llx.StringData(c.Name()),
			"properties":    llx.DictData(props),
			"inventoryPath": llx.StringData(c.InventoryPath),
		})
		if err != nil {
			return nil, err
		}

		mqlClusters[i] = mqlCluster
	}

	return mqlClusters, nil
}

func (v *mqlVsphereCluster) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlVsphereCluster) hosts() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	cluster, err := client.Cluster(path)
	if err != nil {
		return nil, err
	}

	vhosts, err := client.ListHosts(nil, cluster)
	if err != nil {
		return nil, err
	}
	return newVsphereHostResources(client, v.MqlRuntime, vhosts)
}

func (v *mqlVsphereDatacenter) vms() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	dc, err := vClient.Datacenter(path)
	if err != nil {
		return nil, err
	}

	vms, err := vClient.ListVirtualMachines(dc)
	if err != nil {
		return nil, err
	}

	mqlVms := make([]interface{}, len(vms))
	for i, vm := range vms {
		vmInfo, err := resourceclient.VmInfo(vm)
		if err != nil {
			return nil, err
		}

		mqlVm, err := newMqlVm(v.MqlRuntime, vm, vmInfo)
		if err != nil {
			return nil, err
		}

		mqlVms[i] = mqlVm
	}

	return mqlVms, nil
}

func (v *mqlVsphereDatacenter) distributedSwitches() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data
	if path == "" {
		path = "/"
	}

	vswitches, err := client.GetDistributedVirtualSwitches(context.Background(), path)
	if err != nil {
		return nil, err
	}

	mqlVswitches := make([]interface{}, len(vswitches))
	for i, s := range vswitches {

		config, err := client.GetDistributedVirtualSwitchConfig(context.Background(), s)
		if err != nil {
			return nil, err
		}
		configMap, err := resourceclient.DistributedVirtualSwitchConfig(config)
		if err != nil {
			return nil, err
		}

		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vswitch.dvs", map[string]*llx.RawData{
			"moid":       llx.StringData(s.Reference().Encode()),
			"name":       llx.StringData(s.Name()),
			"properties": llx.DictData(configMap),
		})
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		r := mqlVswitch.(*mqlVsphereVswitchDvs)
		r.hostInventoryPath = s.InventoryPath

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereDatacenter) distributedPortGroups() ([]interface{}, error) {
	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := resourceclient.New(conn.Client())

	distPGs, err := client.GetDistributedVirtualPortgroups(context.Background(), path)
	if err != nil {
		return nil, err
	}

	mqlPGs := make([]interface{}, len(distPGs))
	for i, distPG := range distPGs {
		config, err := client.GetDistributedVirtualPortgroupConfig(context.Background(), distPG)
		if err != nil {
			return nil, err
		}

		configMap, err := resourceclient.DistributedVirtualPortgroupConfig(config)
		if err != nil {
			return nil, err
		}

		name := distPG.Name()
		mqlDistPG, err := NewResource(v.MqlRuntime, "vsphere.vswitch.portgroup", map[string]*llx.RawData{
			"moid":       llx.StringData(distPG.Reference().Encode()),
			"name":       llx.StringData(name),
			"properties": llx.DictData(configMap),
		})
		if err != nil {
			return nil, err
		}

		mqlPGs[i] = mqlDistPG.(*mqlVsphereVswitchPortgroup)
	}

	return mqlPGs, nil
}

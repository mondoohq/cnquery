// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/vsphere/connection"
	"go.mondoo.com/mql/v13/providers/vsphere/resources/resourceclient"
	"go.mondoo.com/mql/v13/types"
	"golang.org/x/sync/errgroup"
)

// discoveryConcurrency caps the number of concurrent per-asset SOAP calls we
// fan out during host/VM enumeration. Higher = lower wall-clock for a single
// discovery, but vCenter rate-limits concurrent sessions and the gain
// flattens past ~16 workers.
const discoveryConcurrency = 16

// stagedHost holds the per-host data fetched in the concurrent stage of
// newVsphereHostResources, before the sequential CreateResource pass.
type stagedHost struct {
	hostInfo                *mo.HostSystem
	props                   map[string]any
	name                    string
	tags                    []string
	lockdownMode            string
	firewallIncomingBlocked bool
	firewallOutgoingBlocked bool
	secureBootEnabled       bool
}

func newVsphereHostResources(vClient *resourceclient.Client, runtime *plugin.Runtime, vhosts []*object.HostSystem) ([]any, error) {
	conn := runtime.Connection.(*connection.VsphereConnection)
	ctx := context.Background()

	hostRefs := make([]mo.Reference, len(vhosts))
	for i, h := range vhosts {
		hostRefs[i] = h.Reference()
	}
	vapiTagsByMoid := BatchGetTags(ctx, hostRefs, vClient.Client.Client, conn.Conf)

	// Stage 1 — concurrent fetch of per-host data (SOAP + JSON marshaling).
	// CreateResource is held back to stage 2 because it touches the runtime
	// resource cache, which we keep single-threaded.
	staged := make([]stagedHost, len(vhosts))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(discoveryConcurrency)
	for i, h := range vhosts {
		g.Go(func() error {
			hostInfo, err := resourceclient.HostInfo(gctx, h)
			if err != nil {
				return err
			}
			props, err := resourceclient.HostProperties(hostInfo)
			if err != nil {
				return err
			}

			s := stagedHost{hostInfo: hostInfo, props: props}
			if hostInfo != nil {
				s.name = hostInfo.Name
			}
			if vapi := vapiTagsByMoid[h.Reference().Value]; len(vapi) > 0 {
				s.tags = vapi
			} else if hostInfo != nil {
				s.tags = extractTagKeys(hostInfo.Tag)
			}
			s.lockdownMode, s.firewallIncomingBlocked, s.firewallOutgoingBlocked, s.secureBootEnabled = hostHardeningArgs(hostInfo)
			staged[i] = s
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Stage 2 — sequential CreateResource pass.
	mqlHosts := make([]any, len(vhosts))
	for i, s := range staged {
		h := vhosts[i]
		mqlHost, err := CreateResource(runtime, "vsphere.host", map[string]*llx.RawData{
			"moid":                    llx.StringData(h.Reference().Encode()),
			"name":                    llx.StringData(s.name),
			"properties":              llx.DictData(s.props),
			"inventoryPath":           llx.StringData(h.InventoryPath),
			"tags":                    llx.ArrayData(convert.SliceAnyToInterface(s.tags), types.String),
			"lockdownMode":            llx.StringData(s.lockdownMode),
			"firewallIncomingBlocked": llx.BoolData(s.firewallIncomingBlocked),
			"firewallOutgoingBlocked": llx.BoolData(s.firewallOutgoingBlocked),
			"secureBootEnabled":       llx.BoolData(s.secureBootEnabled),
		})
		if err != nil {
			return nil, err
		}
		mqlHost.(*mqlVsphereHost).host = s.hostInfo
		mqlHosts[i] = mqlHost
	}
	return mqlHosts, nil
}

func (v *mqlVsphereDatacenter) id() (string, error) {
	return v.Moid.Data, nil
}

func (v *mqlVsphereDatacenter) hosts() ([]any, error) {
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

func (v *mqlVsphereDatacenter) clusters() ([]any, error) {
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

	mqlClusters := make([]any, len(vCluster))
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

func (v *mqlVsphereCluster) hosts() ([]any, error) {
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

func (v *mqlVsphereDatacenter) vms() ([]any, error) {
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

	ctx := context.Background()
	vmRefs := make([]mo.Reference, len(vms))
	for i, vm := range vms {
		vmRefs[i] = vm.Reference()
	}
	vapiTagsByMoid := BatchGetTags(ctx, vmRefs, vClient.Client.Client, conn.Conf)

	// Stage 1 — concurrent VmInfo + VmProperties for each VM.
	type stagedVm struct {
		vmInfo *mo.VirtualMachine
		props  map[string]any
		tags   []string
	}
	staged := make([]stagedVm, len(vms))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(discoveryConcurrency)
	for i, vm := range vms {
		g.Go(func() error {
			vmInfo, err := resourceclient.VmInfo(gctx, vm)
			if err != nil {
				return err
			}
			props, err := resourceclient.VmProperties(vmInfo)
			if err != nil {
				return err
			}

			s := stagedVm{vmInfo: vmInfo, props: props}
			if vapi := vapiTagsByMoid[vm.Reference().Value]; len(vapi) > 0 {
				s.tags = vapi
			} else if vmInfo != nil {
				s.tags = extractTagKeys(vmInfo.Tag)
			}
			staged[i] = s
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Stage 2 — sequential CreateResource pass.
	mqlVms := make([]any, len(vms))
	for i, s := range staged {
		vm := vms[i]
		var name string
		if s.vmInfo != nil && s.vmInfo.Config != nil {
			name = s.vmInfo.Config.Name
		}
		mqlVm, err := CreateResource(v.MqlRuntime, "vsphere.vm", map[string]*llx.RawData{
			"moid":          llx.StringData(vm.Reference().Encode()),
			"name":          llx.StringData(name),
			"properties":    llx.DictData(s.props),
			"inventoryPath": llx.StringData(vm.InventoryPath),
			"tags":          llx.ArrayData(convert.SliceAnyToInterface(s.tags), types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlVm.(*mqlVsphereVm).vm = s.vmInfo
		mqlVms[i] = mqlVm
	}
	return mqlVms, nil
}

func (v *mqlVsphereDatacenter) distributedSwitches() ([]any, error) {
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

	mqlVswitches := make([]any, len(vswitches))
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

func (v *mqlVsphereDatacenter) distributedPortGroups() ([]any, error) {
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

	mqlPGs := make([]any, len(distPGs))
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

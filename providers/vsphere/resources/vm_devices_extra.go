// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlVsphereVmCdromInternal struct {
	cacheDatastoreMoid string
}

func (v *mqlVsphereVm) cdroms() ([]any, error) {
	if v.vm == nil || v.vm.Config == nil {
		return []any{}, nil
	}
	vmMoid := v.Moid.Data
	out := []any{}
	for _, dev := range v.vm.Config.Hardware.Device {
		cd, ok := dev.(*types.VirtualCdrom)
		if !ok {
			continue
		}

		var (
			label                                            string
			backingType, isoPath, datastoreMoid              string
			connected, connectedAtPowerOn, allowGuestControl bool
		)
		if cd.DeviceInfo != nil {
			label = cd.DeviceInfo.GetDescription().Label
		}
		if conn := cd.Connectable; conn != nil {
			connected = conn.Connected
			connectedAtPowerOn = conn.StartConnected
			allowGuestControl = conn.AllowGuestControl
		}

		switch b := cd.Backing.(type) {
		case *types.VirtualCdromIsoBackingInfo:
			backingType = "iso"
			isoPath = b.FileName
			if b.Datastore != nil {
				datastoreMoid = b.Datastore.Encode()
			}
		case *types.VirtualCdromAtapiBackingInfo:
			backingType = "atapi"
		case *types.VirtualCdromPassthroughBackingInfo:
			backingType = "passthrough"
		case *types.VirtualCdromRemoteAtapiBackingInfo:
			backingType = "remoteAtapi"
		case *types.VirtualCdromRemotePassthroughBackingInfo:
			backingType = "remotePassthrough"
		}

		res, err := CreateResource(v.MqlRuntime, "vsphere.vm.cdrom", map[string]*llx.RawData{
			"__id":               llx.StringData(vmMoid + "/cdrom/" + strconv.Itoa(int(cd.Key))),
			"key":                llx.IntData(int64(cd.Key)),
			"label":              llx.StringData(label),
			"backingType":        llx.StringData(backingType),
			"isoPath":            llx.StringData(isoPath),
			"connected":          llx.BoolData(connected),
			"connectedAtPowerOn": llx.BoolData(connectedAtPowerOn),
			"allowGuestControl":  llx.BoolData(allowGuestControl),
		})
		if err != nil {
			return nil, err
		}
		mqlCdrom := res.(*mqlVsphereVmCdrom)
		mqlCdrom.cacheDatastoreMoid = datastoreMoid
		out = append(out, mqlCdrom)
	}
	return out, nil
}

func (c *mqlVsphereVmCdrom) datastore() (*mqlVsphereDatastore, error) {
	if c.cacheDatastoreMoid == "" {
		c.Datastore.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	inv, err := loadVsphereInventory(c.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if ds, ok := inv.datastores[c.cacheDatastoreMoid]; ok {
		return ds, nil
	}
	c.Datastore.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (v *mqlVsphereVm) networkAdapters() ([]any, error) {
	if v.vm == nil || v.vm.Config == nil {
		return []any{}, nil
	}
	vmMoid := v.Moid.Data
	out := []any{}
	for _, dev := range v.vm.Config.Hardware.Device {
		nicBase, ok := dev.(types.BaseVirtualEthernetCard)
		if !ok {
			continue
		}
		nic := nicBase.GetVirtualEthernetCard()

		var adapterType string
		switch dev.(type) {
		case *types.VirtualE1000:
			adapterType = "e1000"
		case *types.VirtualE1000e:
			adapterType = "e1000e"
		case *types.VirtualVmxnet3:
			adapterType = "vmxnet3"
		case *types.VirtualVmxnet2:
			adapterType = "vmxnet2"
		case *types.VirtualVmxnet:
			adapterType = "vmxnet"
		case *types.VirtualPCNet32:
			adapterType = "pcnet32"
		case *types.VirtualSriovEthernetCard:
			adapterType = "sriov"
		}

		var (
			label                                            string
			connected, connectedAtPowerOn, allowGuestControl bool
			wakeOnLan                                        bool
			backingType, portGroupName, portGroupMoid        string
		)
		if nic.DeviceInfo != nil {
			label = nic.DeviceInfo.GetDescription().Label
		}
		if conn := nic.Connectable; conn != nil {
			connected = conn.Connected
			connectedAtPowerOn = conn.StartConnected
			allowGuestControl = conn.AllowGuestControl
		}
		if nic.WakeOnLanEnabled != nil {
			wakeOnLan = *nic.WakeOnLanEnabled
		}

		switch b := nic.Backing.(type) {
		case *types.VirtualEthernetCardNetworkBackingInfo:
			backingType = "network"
			portGroupName = b.DeviceName
		case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
			backingType = "dvs"
			portGroupMoid = b.Port.PortgroupKey
		case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
			backingType = "opaque"
			portGroupName = b.OpaqueNetworkId
		case *types.VirtualEthernetCardLegacyNetworkBackingInfo:
			backingType = "network"
			portGroupName = b.DeviceName
		}

		res, err := CreateResource(v.MqlRuntime, "vsphere.vm.networkAdapter", map[string]*llx.RawData{
			"__id":               llx.StringData(vmMoid + "/nic/" + strconv.Itoa(int(nic.Key))),
			"key":                llx.IntData(int64(nic.Key)),
			"label":              llx.StringData(label),
			"adapterType":        llx.StringData(adapterType),
			"macAddress":         llx.StringData(nic.MacAddress),
			"addressType":        llx.StringData(nic.AddressType),
			"connected":          llx.BoolData(connected),
			"connectedAtPowerOn": llx.BoolData(connectedAtPowerOn),
			"allowGuestControl":  llx.BoolData(allowGuestControl),
			"wakeOnLan":          llx.BoolData(wakeOnLan),
			"backingType":        llx.StringData(backingType),
			"portGroupName":      llx.StringData(portGroupName),
			"portGroupMoid":      llx.StringData(portGroupMoid),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (v *mqlVsphereVm) cpuAllocation() (*mqlVsphereVmCpuAllocation, error) {
	if v.vm == nil || v.vm.Config == nil || v.vm.Config.CpuAllocation == nil {
		v.CpuAllocation.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := buildAllocationResource(v.MqlRuntime, "vsphere.vm.cpuAllocation",
		v.Moid.Data+"/cpuAllocation", v.vm.Config.CpuAllocation, false)
	if err != nil {
		return nil, err
	}
	return res.(*mqlVsphereVmCpuAllocation), nil
}

func (v *mqlVsphereVm) memoryAllocation() (*mqlVsphereVmMemoryAllocation, error) {
	if v.vm == nil || v.vm.Config == nil || v.vm.Config.MemoryAllocation == nil {
		v.MemoryAllocation.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := buildAllocationResource(v.MqlRuntime, "vsphere.vm.memoryAllocation",
		v.Moid.Data+"/memoryAllocation", v.vm.Config.MemoryAllocation, true)
	if err != nil {
		return nil, err
	}
	return res.(*mqlVsphereVmMemoryAllocation), nil
}

// buildAllocationResource creates the cpu/memory allocation resource. When
// memory is true, the field names use the MB suffix (and overheadLimitMB is
// included); otherwise MHz suffix and overheadLimit is omitted.
func buildAllocationResource(runtime *plugin.Runtime, resName, id string, ra *types.ResourceAllocationInfo, memory bool) (plugin.Resource, error) {
	var reservation, limit int64 = 0, -1
	if ra.Reservation != nil {
		reservation = *ra.Reservation
	}
	if ra.Limit != nil {
		limit = *ra.Limit
	}
	expandable := false
	if ra.ExpandableReservation != nil {
		expandable = *ra.ExpandableReservation
	}
	var sharesLevel string
	var shares int64
	if ra.Shares != nil {
		sharesLevel = string(ra.Shares.Level)
		shares = int64(ra.Shares.Shares)
	}

	args := map[string]*llx.RawData{
		"__id":                  llx.StringData(id),
		"expandableReservation": llx.BoolData(expandable),
		"sharesLevel":           llx.StringData(sharesLevel),
		"shares":                llx.IntData(shares),
	}
	if memory {
		var overhead int64 = -1
		if ra.OverheadLimit != nil {
			overhead = *ra.OverheadLimit
		}
		args["reservationMB"] = llx.IntData(reservation)
		args["limitMB"] = llx.IntData(limit)
		args["overheadLimitMB"] = llx.IntData(overhead)
	} else {
		args["reservationMHz"] = llx.IntData(reservation)
		args["limitMHz"] = llx.IntData(limit)
	}
	return CreateResource(runtime, resName, args)
}

// portGroup resolves the DVS port group the adapter is connected to. Returns
// null for standard portgroup or opaque-network backings (which have no
// portGroupMoid) and for DVS backings whose port group isn't in inventory.
func (n *mqlVsphereVmNetworkAdapter) portGroup() (*mqlVsphereVswitchPortgroup, error) {
	moid := n.PortGroupMoid.Data
	if moid == "" {
		n.PortGroup.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	inv, err := loadVsphereInventory(n.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if pg, ok := inv.dvPortGroups[moid]; ok {
		return pg, nil
	}
	n.PortGroup.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

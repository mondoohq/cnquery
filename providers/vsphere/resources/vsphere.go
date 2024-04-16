// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/vsphere/connection"
	"go.mondoo.com/cnquery/v11/providers/vsphere/resources/resourceclient"
)

func getClientInstance(conn *connection.VsphereConnection) *resourceclient.Client {
	return resourceclient.New(conn.Client())
}

func esxiClient(conn *connection.VsphereConnection, path string) (*resourceclient.Esxi, error) {
	vClient := getClientInstance(conn)

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	esxi := resourceclient.NewEsxiClient(vClient.Client, path, host)
	return esxi, nil
}

func (v *mqlVsphere) id() (string, error) {
	return "vsphere", nil
}

func (v *mqlVsphere) about() (map[string]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	return client.AboutInfo()
}

func (v *mqlVsphere) datacenters() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	// fetch datacenters
	dcs, err := client.ListDatacenters()
	if err != nil {
		return nil, err
	}

	// convert datacenter to MQL
	datacenters := make([]interface{}, len(dcs))
	for i, dc := range dcs {
		mqlDc, err := CreateResource(v.MqlRuntime, "vsphere.datacenter", map[string]*llx.RawData{
			"moid":          llx.StringData(dc.Reference().Encode()),
			"name":          llx.StringData(dc.Name()),
			"inventoryPath": llx.StringData(dc.InventoryPath),
		})
		if err != nil {
			return nil, err
		}

		datacenters[i] = mqlDc
	}

	return datacenters, nil
}

func (v *mqlVsphere) licenses() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	client := getClientInstance(conn)

	// fetch license
	lcs, err := client.ListLicenses()
	if err != nil {
		return nil, err
	}

	// convert licenses to MQL
	licenses := make([]interface{}, len(lcs))
	for i, l := range lcs {
		mqlLicense, err := CreateResource(v.MqlRuntime, "vsphere.license", map[string]*llx.RawData{
			"name":  llx.StringData(l.Name),
			"total": llx.IntData(int64(l.Total)),
			"used":  llx.IntData(int64(l.Used)),
		})
		if err != nil {
			return nil, err
		}

		licenses[i] = mqlLicense
	}

	return licenses, nil
}

func (v *mqlEsxi) id() (string, error) {
	return "esxi", nil
}

func esxiHostProperties(conn *connection.VsphereConnection) (*object.HostSystem, *mo.HostSystem, error) {
	var h *object.HostSystem
	vClient := conn.Client()
	cl := resourceclient.New(vClient)
	if !vClient.IsVC() {
		// ESXi connections only have one host
		dcs, err := cl.ListDatacenters()
		if err != nil {
			return nil, nil, err
		}

		if len(dcs) != 1 {
			return nil, nil, errors.New("could not find single esxi datacenter")
		}

		dc := dcs[0]

		hosts, err := cl.ListHosts(dc, nil)
		if err != nil {
			return nil, nil, err
		}

		if len(hosts) != 1 {
			return nil, nil, errors.New("could not find single esxi host")
		}

		h = hosts[0]
	} else {
		// check if the connection was initialized with a specific host
		identifier, err := conn.Identifier()
		if err != nil || !connection.IsVsphereResourceID(identifier) {
			return nil, nil, errors.New("esxi resource is only supported for esxi connections or vsphere vm connections")
		}

		// extract type and inventory
		moid, err := connection.ParseVsphereResourceID(identifier)
		if err != nil {
			return nil, nil, err
		}

		if moid.Type != "HostSystem" {
			return nil, nil, errors.New("esxi resource is not supported for vsphere type " + moid.Type)
		}

		h, err = cl.HostByMoid(moid)
		if err != nil {
			return nil, nil, errors.New("could not find the esxi host via platform id: " + identifier)
		}
	}

	// todo sync with GetHosts
	hostInfo, err := resourceclient.HostInfo(h)
	if err != nil {
		return nil, nil, err
	}

	return h, hostInfo, nil
}

// GetHost returns the information about the current ESXi host
// Deprecated: use vsphere.host resource instead
func (v *mqlEsxi) host() (*mqlVsphereHost, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	h, hostInfo, err := esxiHostProperties(conn)
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

	mqlHost, err := CreateResource(v.MqlRuntime, "vsphere.host", map[string]*llx.RawData{
		"moid":          llx.StringData(h.Reference().Encode()),
		"name":          llx.StringData(name),
		"properties":    llx.DictData(props),
		"inventoryPath": llx.StringData(h.InventoryPath),
	})
	if err != nil {
		return nil, err
	}
	return mqlHost.(*mqlVsphereHost), nil
}

func esxiVmProperties(conn *connection.VsphereConnection) (*object.VirtualMachine, *mo.VirtualMachine, error) {
	vClient := conn.Client()
	cl := resourceclient.New(vClient)

	// check if the connection was initialized with a specific host
	identifier, err := conn.Identifier()
	if err != nil || !connection.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("esxi resource is only supported for esxi connections or vsphere vm connections")
	}

	// extract type and inventory
	moid, err := connection.ParseVsphereResourceID(identifier)
	if err != nil {
		return nil, nil, err
	}

	if moid.Type != "VirtualMachine" {
		return nil, nil, errors.New("esxi resource is not supported for vsphere type " + moid.Type)
	}

	vm, err := cl.VirtualMachineByMoid(moid)
	if err != nil {
		return nil, nil, errors.New("could not find the esxi vm via platform id: " + identifier)
	}

	vmInfo, err := resourceclient.VmInfo(vm)
	if err != nil {
		return nil, nil, err
	}

	return vm, vmInfo, nil
}

func (v *mqlEsxi) vm() (*mqlVsphereVm, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)

	vm, vmInfo, err := esxiVmProperties(conn)
	if err != nil {
		return nil, err
	}

	return newMqlVm(v.MqlRuntime, vm, vmInfo)
}

func (v *mqlEsxiCommand) id() (string, error) {
	return v.Command.Data, nil
}

func initEsxiCommand(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.VsphereConnection)

	if len(args) > 2 {
		return args, nil, nil
	}

	// check if the command arg is provided
	commandRaw := args["command"]
	if commandRaw == nil {
		return args, nil, nil
	}

	// check if the connection was initialized with a specific host
	identifier, err := conn.Identifier()
	if err != nil || !connection.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("could not determine inventoryPath from provider connection")
	}

	h, err := hostSystem(conn, identifier)
	if err != nil {
		return nil, nil, err
	}

	args["inventoryPath"] = llx.StringData(h.InventoryPath)
	return args, nil, nil
}

func hostSystem(conn *connection.VsphereConnection, identifier string) (*object.HostSystem, error) {
	var h *object.HostSystem
	vClient := conn.Client()
	cl := resourceclient.New(vClient)

	// extract type and inventory
	moid, err := connection.ParseVsphereResourceID(identifier)
	if err != nil {
		return nil, err
	}

	if moid.Type != "HostSystem" {
		return nil, errors.New("ESXi resource is not supported for vsphere type " + moid.Type)
	}

	h, err = cl.HostByMoid(moid)
	if err != nil {
		return nil, errors.New("could not find the esxi host via platform id: " + identifier)
	}

	return h, nil
}

func (v *mqlEsxiCommand) result() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	esxiClient, err := esxiClient(conn, path)
	if err != nil {
		return nil, err
	}

	if v.Command.Error != nil {
		return nil, v.Command.Error
	}
	cmd := v.Command.Data

	res := []interface{}{}

	resp, err := esxiClient.Command(cmd)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		res = append(res, resp[i])
	}

	return res, nil
}

func (v *mqlVsphereLicense) id() (string, error) {
	return v.Name.Data, nil
}

func (v *mqlVsphereVmknic) id() (string, error) {
	return v.Name.Data, nil
}

func (v *mqlEsxiVib) id() (string, error) {
	return v.Id.Data, nil
}

func (v *mqlEsxiKernelmodule) id() (string, error) {
	return v.Name.Data, nil
}

func (v *mqlEsxiService) id() (string, error) {
	return v.Key.Data, nil
}

func (v *mqlEsxiTimezone) id() (string, error) {
	return v.Key.Data, nil
}

func (v *mqlEsxiNtpconfig) id() (string, error) {
	return v.Id.Data, nil
}

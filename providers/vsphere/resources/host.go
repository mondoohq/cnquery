// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"github.com/vmware/govmomi/vim25/mo"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/vsphere/connection"
	"go.mondoo.com/cnquery/v11/providers/vsphere/resources/resourceclient"
	"go.mondoo.com/cnquery/v11/types"
)

type mqlVsphereHostInternal struct {
	host *mo.HostSystem
}

func (v *mqlVsphereHost) id() (string, error) {
	return v.Moid.Data, nil
}

func initVsphereHost(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.VsphereConnection)
	h, hostInfo, err := esxiHostProperties(conn)
	if err != nil {
		return nil, nil, err
	}

	props, err := resourceclient.HostProperties(hostInfo)
	if err != nil {
		return nil, nil, err
	}

	var name string
	if hostInfo != nil {
		name = hostInfo.Name
	}

	args["moid"] = llx.StringData(h.Reference().Encode())
	args["name"] = llx.StringData(name)
	args["properties"] = llx.DictData(props)
	args["inventoryPath"] = llx.StringData(h.InventoryPath)

	return args, nil, nil
}

func (v *mqlVsphereHost) esxiClient() (*resourceclient.Esxi, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	return esxiClient(conn, path)
}

func (v *mqlVsphereHost) standardSwitch() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	vswitches, err := esxiClient.VswitchStandard()
	if err != nil {
		return nil, err
	}

	mqlVswitches := make([]interface{}, len(vswitches))
	for i, s := range vswitches {
		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vswitch.standard", map[string]*llx.RawData{
			"name":       llx.StringData(s["Name"].(string)),
			"properties": llx.DictData(s),
		})
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		r := mqlVswitch.(*mqlVsphereVswitchStandard)
		r.hostInventoryPath = esxiClient.InventoryPath
		r.parentResource = v

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereHost) distributedSwitch() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	vswitches, err := esxiClient.VswitchDvs()
	if err != nil {
		return nil, err
	}

	mqlVswitches := make([]interface{}, len(vswitches))
	for i, s := range vswitches {
		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vswitch.dvs", map[string]*llx.RawData{
			"name":       llx.StringData(s["Name"].(string)),
			"properties": llx.DictData(s),
		})
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		r := mqlVswitch.(*mqlVsphereVswitchDvs)
		r.hostInventoryPath = esxiClient.InventoryPath
		r.parentResource = v

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereHost) adapters() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	adapters, err := esxiClient.Adapters()
	if err != nil {
		return nil, err
	}

	pParams, err := esxiClient.ListNicPauseParams()
	if err != nil {
		return nil, err
	}

	pauseParams := map[string]map[string]interface{}{}
	// sort pause params by nic
	for i, p := range pParams {
		nicName := pParams[i]["NIC"].(string)
		pauseParams[nicName] = p
	}

	mqlAdapters := make([]interface{}, len(adapters))
	for i, a := range adapters {
		nicName := a["Name"].(string)
		pParams := pauseParams[nicName]

		mqlAdapter, err := CreateResource(v.MqlRuntime, "vsphere.vmnic", map[string]*llx.RawData{
			"name":        llx.StringData(nicName),
			"properties":  llx.DictData(a),
			"pauseParams": llx.DictData(pParams),
		})
		if err != nil {
			return nil, err
		}

		// set inventory path
		r := mqlAdapter.(*mqlVsphereVmnic)
		r.hostInventoryPath = esxiClient.InventoryPath

		mqlAdapters[i] = mqlAdapter
	}

	return mqlAdapters, nil
}

func (v *mqlVsphereVmnic) id() (string, error) {
	return v.Name.Data, nil
}

type mqlVsphereVmnicInternal struct {
	hostInventoryPath string
}

func (v *mqlVsphereVmnic) esxiClient() (*resourceclient.Esxi, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	return esxiClient(conn, v.hostInventoryPath)
}

func (v *mqlVsphereVmnic) details() (map[string]interface{}, error) {
	name := v.Name.Data

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.ListNicDetails(name)
}

func (v *mqlVsphereHost) vmknics() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	vmknics, err := esxiClient.Vmknics()
	if err != nil {
		return nil, err
	}

	mqlVmknics := make([]interface{}, len(vmknics))
	for i := range vmknics {
		entry := vmknics[i]
		mqlVswitch, err := CreateResource(v.MqlRuntime, "vsphere.vmknic", map[string]*llx.RawData{
			"name":       llx.StringData(entry.Properties["Name"].(string)),
			"properties": llx.DictData(entry.Properties),
			"ipv4":       llx.ArrayData(entry.Ipv4, types.Dict),
			"ipv6":       llx.ArrayData(entry.Ipv6, types.Dict),
			"tags":       llx.ArrayData(convert.SliceAnyToInterface(entry.Tags), types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlVmknics[i] = mqlVswitch
	}

	return mqlVmknics, nil
}

func (v *mqlVsphereHost) packages() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	vibs, err := esxiClient.Vibs()
	if err != nil {
		return nil, err
	}

	mqlPackages := make([]interface{}, len(vibs))
	for i := range vibs {
		vib := vibs[i]

		// parse timestamps in format "2020-07-16"
		format := "2006-01-02"
		var creationDate *time.Time
		parsedCreation, err := time.Parse(format, vib.CreationDate)
		if err != nil {
			return nil, errors.New("cannot parse vib creationDate: " + vib.CreationDate)
		}
		creationDate = &parsedCreation

		var installDate *time.Time
		parsedInstall, err := time.Parse(format, vib.InstallDate)
		if err != nil {
			return nil, errors.New("cannot parse vib installDate: " + vib.InstallDate)
		}
		installDate = &parsedInstall

		mqlVib, err := CreateResource(v.MqlRuntime, "esxi.vib", map[string]*llx.RawData{
			"id":              llx.StringData(vib.ID),
			"name":            llx.StringData(vib.Name),
			"acceptanceLevel": llx.StringData(vib.AcceptanceLevel),
			"creationDate":    llx.TimeDataPtr(creationDate),
			"installDate":     llx.TimeDataPtr(installDate),
			"status":          llx.StringData(vib.Status),
			"vendor":          llx.StringData(vib.Vendor),
			"version":         llx.StringData(vib.Version),
		})
		if err != nil {
			return nil, err
		}
		mqlPackages[i] = mqlVib
	}

	return mqlPackages, nil
}

func (v *mqlVsphereHost) acceptanceLevel() (string, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return "", err
	}
	return esxiClient.SoftwareAcceptance()
}

func (v *mqlVsphereHost) kernelModules() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	modules, err := esxiClient.KernelModules()
	if err != nil {
		return nil, err
	}

	mqlModules := make([]interface{}, len(modules))
	for i, m := range modules {
		mqlModule, err := CreateResource(v.MqlRuntime, "esxi.kernelmodule", map[string]*llx.RawData{
			"name":                 llx.StringData(m.Module),
			"modulefile":           llx.StringData(m.ModuleFile),
			"version":              llx.StringData(m.Version),
			"loaded":               llx.BoolData(m.Loaded),
			"license":              llx.StringData(m.License),
			"enabled":              llx.BoolData(m.Enabled),
			"signedStatus":         llx.StringData(m.SignedStatus),
			"signatureDigest":      llx.StringData(m.SignatureDigest),
			"signatureFingerprint": llx.StringData(m.SignatureFingerPrint),
			"vibAcceptanceLevel":   llx.StringData(m.VIBAcceptanceLevel),
		})
		if err != nil {
			return nil, err
		}
		mqlModules[i] = mqlModule
	}

	return mqlModules, nil
}

func (v *mqlVsphereHost) advancedSettings() (map[string]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return resourceclient.HostOptions(host)
}

func (v *mqlVsphereHost) services() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	services, err := resourceclient.HostServices(host)
	if err != nil {
		return nil, err
	}
	mqlServices := make([]interface{}, len(services))
	for i, s := range services {
		mqlService, err := CreateResource(v.MqlRuntime, "esxi.service", map[string]*llx.RawData{
			"key":           llx.StringData(s.Key),
			"label":         llx.StringData(s.Label),
			"required":      llx.BoolData(s.Required),
			"uninstallable": llx.BoolData(s.Uninstallable),
			"running":       llx.BoolData(s.Running),
			"ruleset":       llx.ArrayData(convert.SliceAnyToInterface(s.Ruleset), types.String),
			"policy":        llx.StringData(s.Policy), // on, off, automatic
		})
		if err != nil {
			return nil, err
		}
		mqlServices[i] = mqlService
	}
	return mqlServices, nil
}

func (v *mqlVsphereHost) timezone() (*mqlEsxiTimezone, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	datetimeinfo, err := resourceclient.HostDateTime(host)
	if err != nil {
		return nil, err
	}

	if datetimeinfo == nil {
		return nil, errors.New("vsphere does not return HostDateTimeSystem timezone information")
	}

	mqlTimezone, err := CreateResource(v.MqlRuntime, "esxi.timezone", map[string]*llx.RawData{
		"key":         llx.StringData(datetimeinfo.TimeZone.Key),
		"name":        llx.StringData(datetimeinfo.TimeZone.Name),
		"offset":      llx.IntData(int64(datetimeinfo.TimeZone.GmtOffset)),
		"description": llx.StringData(datetimeinfo.TimeZone.Description),
	})
	if err != nil {
		return nil, err
	}

	return mqlTimezone.(*mqlEsxiTimezone), nil
}

func (v *mqlVsphereHost) ntp() (*mqlEsxiNtpconfig, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	vClient := getClientInstance(conn)

	if v.InventoryPath.Error != nil {
		return nil, v.InventoryPath.Error
	}
	path := v.InventoryPath.Data

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	datetimeinfo, err := resourceclient.HostDateTime(host)
	if err != nil {
		return nil, err
	}

	var server []interface{}
	var config []interface{}

	if datetimeinfo != nil && datetimeinfo.NtpConfig != nil {
		server = convert.SliceAnyToInterface(datetimeinfo.NtpConfig.Server)
		config = convert.SliceAnyToInterface(datetimeinfo.NtpConfig.ConfigFile)
	}

	mqlNtpConfig, err := CreateResource(v.MqlRuntime, "esxi.ntpconfig", map[string]*llx.RawData{
		"id":     llx.StringData("ntp/" + host.InventoryPath),
		"server": llx.ArrayData(server, types.String),
		"config": llx.ArrayData(config, types.String),
	})
	if err != nil {
		return nil, err
	}

	return mqlNtpConfig.(*mqlEsxiNtpconfig), nil
}

func (v *mqlVsphereHost) snmp() (map[string]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.Snmp()
}

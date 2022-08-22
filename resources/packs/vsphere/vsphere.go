package vsphere

import (
	"errors"
	"reflect"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"go.mondoo.io/mondoo/motor/providers"
	provider "go.mondoo.io/mondoo/motor/providers/vsphere"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/vsphere/info"
	"go.mondoo.io/mondoo/resources/packs/vsphere/resourceclient"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func getClientInstance(t providers.Instance) (*resourceclient.Client, error) {
	vt, ok := t.(*provider.Provider)
	if !ok {
		return nil, errors.New("vsphere resource is not supported on this transport")
	}

	cl := resourceclient.New(vt.Client())
	return cl, nil
}

func esxiClient(t providers.Instance, path string) (*resourceclient.Esxi, error) {
	vClient, err := getClientInstance(t)
	if err != nil {
		return nil, err
	}

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	esxi := resourceclient.NewEsxiClient(vClient.Client, path, host)
	return esxi, nil
}

func (v *mqlVsphereLicense) id() (string, error) {
	return v.Name()
}

func (v *mqlVsphereVmknic) id() (string, error) {
	return v.Name()
}

func (v *mqlEsxiVib) id() (string, error) {
	return v.Id()
}

func (v *mqlEsxiKernelmodule) id() (string, error) {
	return v.Name()
}

func (v *mqlEsxiService) id() (string, error) {
	return v.Key()
}

func (v *mqlEsxiTimezone) id() (string, error) {
	return v.Key()
}

func (v *mqlEsxiNtpconfig) id() (string, error) {
	return v.Id()
}

func (v *mqlVsphere) id() (string, error) {
	return "vsphere", nil
}

func (v *mqlVsphere) GetAbout() (map[string]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	return client.AboutInfo()
}

func (v *mqlVsphere) GetDatacenters() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// fetch datacenters
	dcs, err := client.ListDatacenters()
	if err != nil {
		return nil, err
	}

	// convert datacenter to MQL
	datacenters := make([]interface{}, len(dcs))
	for i, dc := range dcs {
		mqlDc, err := v.MotorRuntime.CreateResource("vsphere.datacenter",
			"moid", dc.Reference().Encode(),
			"name", dc.Name(),
			"inventoryPath", dc.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		datacenters[i] = mqlDc
	}

	return datacenters, nil
}

func (v *mqlVsphere) GetLicenses() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// fetch license
	lcs, err := client.ListLicenses()
	if err != nil {
		return nil, err
	}

	// convert licenses to MQL
	licenses := make([]interface{}, len(lcs))
	for i, l := range lcs {
		mqlLicense, err := v.MotorRuntime.CreateResource("vsphere.license",
			"name", l.Name,
			"total", int64(l.Total),
			"used", int64(l.Used),
		)
		if err != nil {
			return nil, err
		}

		licenses[i] = mqlLicense
	}

	return licenses, nil
}

func vsphereHosts(vClient *resourceclient.Client, runtime *resources.Runtime, vhosts []*object.HostSystem) ([]interface{}, error) {
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

		mqlHost, err := runtime.CreateResource("vsphere.host",
			"moid", h.Reference().Encode(),
			"name", name,
			"properties", props,
			"inventoryPath", h.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		mqlHosts[i] = mqlHost
	}

	return mqlHosts, nil
}

func (v *mqlVsphereDatacenter) id() (string, error) {
	return v.Moid()
}

func (v *mqlVsphereDatacenter) GetHosts() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	dc, err := client.Datacenter(path)
	if err != nil {
		return nil, err
	}

	vhosts, err := client.ListHosts(dc, nil)
	if err != nil {
		return nil, err
	}
	return vsphereHosts(client, v.MotorRuntime, vhosts)
}

func (v *mqlVsphereDatacenter) GetClusters() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

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

		mqlCluster, err := v.MotorRuntime.CreateResource("vsphere.cluster",
			"moid", c.Reference().Encode(),
			"name", c.Name(),
			"properties", props,
			"inventoryPath", c.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		mqlClusters[i] = mqlCluster
	}

	return mqlClusters, nil
}

func (v *mqlVsphereCluster) id() (string, error) {
	return v.Moid()
}

func (v *mqlVsphereCluster) GetHosts() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	cluster, err := client.Cluster(path)
	if err != nil {
		return nil, err
	}

	vhosts, err := client.ListHosts(nil, cluster)
	if err != nil {
		return nil, err
	}
	return vsphereHosts(client, v.MotorRuntime, vhosts)
}

func (v *mqlVsphereHost) id() (string, error) {
	return v.Moid()
}

func (v *mqlVsphereHost) init(args *resources.Args) (*resources.Args, VsphereHost, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	h, hostInfo, err := esxiHostProperties(v.MotorRuntime)
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

	(*args)["moid"] = h.Reference().Encode()
	(*args)["name"] = name
	(*args)["properties"] = props
	(*args)["inventoryPath"] = h.InventoryPath

	return args, nil, nil
}

func (v *mqlVsphereHost) esxiClient() (*resourceclient.Esxi, error) {
	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	return esxiClient(v.MotorRuntime.Motor.Provider, path)
}

func (v *mqlVsphereHost) GetStandardSwitch() ([]interface{}, error) {
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
		mqlVswitch, err := v.MotorRuntime.CreateResource("vsphere.vswitch.standard",
			"name", s["Name"],
			"properties", s,
		)
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		mqlVswitch.MqlResource().Cache.Store("_host_inventory_path", &resources.CacheEntry{Data: esxiClient.InventoryPath})
		mqlVswitch.MqlResource().Cache.Store("_parent_resource", &resources.CacheEntry{Data: v})

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereHost) GetDistributedSwitch() ([]interface{}, error) {
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
		mqlVswitch, err := v.MotorRuntime.CreateResource("vsphere.vswitch.dvs",
			"name", s["Name"],
			"properties", s,
		)
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		mqlVswitch.MqlResource().Cache.Store("_host_inventory_path", &resources.CacheEntry{Data: esxiClient.InventoryPath})
		mqlVswitch.MqlResource().Cache.Store("_parent_resource", &resources.CacheEntry{Data: v})

		mqlVswitches[i] = mqlVswitch
	}

	return mqlVswitches, nil
}

func (v *mqlVsphereHost) GetAdapters() ([]interface{}, error) {
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

		mqlAdapter, err := v.MotorRuntime.CreateResource("vsphere.vmnic",
			"name", nicName,
			"properties", a,
			"pauseParams", pParams,
		)
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		mqlAdapter.MqlResource().Cache.Store("_host_inventory_path", &resources.CacheEntry{Data: esxiClient.InventoryPath})
		mqlAdapters[i] = mqlAdapter
	}

	return mqlAdapters, nil
}

func (v *mqlVsphereVmnic) id() (string, error) {
	return v.Name()
}

func (v *mqlVsphereVmnic) esxiClient() (*resourceclient.Esxi, error) {
	c, ok := v.MqlResource().Cache.Load("_host_inventory_path")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}
	inventoryPath := c.Data.(string)
	return esxiClient(v.MotorRuntime.Motor.Provider, inventoryPath)
}

func (v *mqlVsphereVmnic) GetDetails() (map[string]interface{}, error) {
	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.ListNicDetails(name)
}

func (v *mqlVsphereHost) GetVmknics() ([]interface{}, error) {
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
		mqlVswitch, err := v.MotorRuntime.CreateResource("vsphere.vmknic",
			"name", entry.Properties["Name"],
			"properties", entry.Properties,
			"ipv4", entry.Ipv4,
			"ipv6", entry.Ipv6,
			"tags", core.StrSliceToInterface(entry.Tags),
		)
		if err != nil {
			return nil, err
		}
		mqlVmknics[i] = mqlVswitch
	}

	return mqlVmknics, nil
}

func (v *mqlVsphereHost) GetPackages() ([]interface{}, error) {
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

		mqlVib, err := v.MotorRuntime.CreateResource("esxi.vib",
			"id", vib.ID,
			"name", vib.Name,
			"acceptanceLevel", vib.AcceptanceLevel,
			"creationDate", creationDate,
			"installDate", installDate,
			"status", vib.Status,
			"vendor", vib.Vendor,
			"version", vib.Version,
		)
		if err != nil {
			return nil, err
		}
		mqlPackages[i] = mqlVib
	}

	return mqlPackages, nil
}

func (v *mqlVsphereHost) GetAcceptanceLevel() (string, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return "", err
	}
	return esxiClient.SoftwareAcceptance()
}

func (v *mqlVsphereHost) GetKernelModules() ([]interface{}, error) {
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
		mqlModule, err := v.MotorRuntime.CreateResource("esxi.kernelmodule",
			"name", m.Module,
			"modulefile", m.ModuleFile,
			"version", m.Version,
			"loaded", m.Loaded,
			"license", m.License,
			"enabled", m.Enabled,
			"signedStatus", m.SignedStatus,
			"signatureDigest", m.SignatureDigest,
			"signatureFingerprint", m.SignatureFingerPrint,
			"vibAcceptanceLevel", m.VIBAcceptanceLevel,
		)
		if err != nil {
			return nil, err
		}
		mqlModules[i] = mqlModule
	}

	return mqlModules, nil
}

func (v *mqlVsphereHost) GetAdvancedSettings() (map[string]interface{}, error) {
	vClient, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	host, err := vClient.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return resourceclient.HostOptions(host)
}

func (v *mqlVsphereHost) GetServices() ([]interface{}, error) {
	vClient, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

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
		mqlService, err := v.MotorRuntime.CreateResource("esxi.service",
			"key", s.Key,
			"label", s.Label,
			"required", s.Required,
			"uninstallable", s.Uninstallable,
			"running", s.Running,
			"ruleset", core.StrSliceToInterface(s.Ruleset),
			"policy", s.Policy, // on, off, automatic
		)
		if err != nil {
			return nil, err
		}
		mqlServices[i] = mqlService
	}
	return mqlServices, nil
}

func (v *mqlVsphereHost) GetTimezone() (interface{}, error) {
	vClient, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

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

	mqlTimezone, err := v.MotorRuntime.CreateResource("esxi.timezone",
		"key", datetimeinfo.TimeZone.Key,
		"name", datetimeinfo.TimeZone.Name,
		"offset", int64(datetimeinfo.TimeZone.GmtOffset),
		"description", datetimeinfo.TimeZone.Description,
	)
	if err != nil {
		return nil, err
	}

	return mqlTimezone, nil
}

func (v *mqlVsphereHost) GetNtp() (interface{}, error) {
	vClient, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

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
		server = core.StrSliceToInterface(datetimeinfo.NtpConfig.Server)
		config = core.StrSliceToInterface(datetimeinfo.NtpConfig.ConfigFile)
	}

	mqlNtpConfig, err := v.MotorRuntime.CreateResource("esxi.ntpconfig",
		"id", "ntp "+host.InventoryPath,
		"server", server,
		"config", config,
	)
	if err != nil {
		return nil, err
	}

	return mqlNtpConfig, nil
}

func (v *mqlVsphereHost) GetSnmp() (map[string]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.Snmp()
}

func (v *mqlVsphereDatacenter) GetVms() ([]interface{}, error) {
	vClient, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

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

		props, err := resourceclient.VmProperties(vmInfo)
		if err != nil {
			return nil, err
		}

		var name string
		if vmInfo != nil && vmInfo.Config != nil {
			name = vmInfo.Config.Name
		}

		mqlVm, err := v.MotorRuntime.CreateResource("vsphere.vm",
			"moid", vm.Reference().Encode(),
			"name", name,
			"properties", props,
			"inventoryPath", vm.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		mqlVms[i] = mqlVm
	}

	return mqlVms, nil
}

func (v *mqlVsphereVm) id() (string, error) {
	return v.Moid()
}

func (v *mqlVsphereVm) GetAdvancedSettings() (map[string]interface{}, error) {
	vClient, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	vm, err := vClient.VirtualMachineByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return resourceclient.AdvancedSettings(vm)
}

func (v *mqlEsxi) id() (string, error) {
	return "esxi", nil
}

func esxiHostProperties(runtime *resources.Runtime) (*object.HostSystem, *mo.HostSystem, error) {
	t := runtime.Motor.Provider
	vt, ok := t.(*provider.Provider)
	if !ok {
		return nil, nil, errors.New("esxi resource is not supported on this transport")
	}

	var h *object.HostSystem
	vClient := vt.Client()
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
		identifier, err := vt.Identifier()
		if err != nil || !provider.IsVsphereResourceID(identifier) {
			return nil, nil, errors.New("esxi resource is only supported for esxi connections or vsphere vm connections")
		}

		// extract type and inventory
		moid, err := provider.ParseVsphereResourceID(identifier)
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
func (v *mqlEsxi) GetHost() (interface{}, error) {
	h, hostInfo, err := esxiHostProperties(v.MotorRuntime)
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

	mqlHost, err := v.MotorRuntime.CreateResource("vsphere.host",
		"moid", h.Reference().Encode(),
		"name", name,
		"properties", props,
		"inventoryPath", h.InventoryPath,
	)
	if err != nil {
		return nil, err
	}
	return mqlHost, nil
}

func esxiVmProperties(runtime *resources.Runtime) (*object.VirtualMachine, *mo.VirtualMachine, error) {
	t := runtime.Motor.Provider
	vt, ok := t.(*provider.Provider)
	if !ok {
		return nil, nil, errors.New("esxi resource is not supported on this transport")
	}

	vClient := vt.Client()
	cl := resourceclient.New(vClient)

	// check if the connection was initialized with a specific host
	identifier, err := vt.Identifier()
	if err != nil || !provider.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("esxi resource is only supported for esxi connections or vsphere vm connections")
	}

	// extract type and inventory
	moid, err := provider.ParseVsphereResourceID(identifier)
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

func (v *mqlEsxi) GetVm() (interface{}, error) {
	vm, vmInfo, err := esxiVmProperties(v.MotorRuntime)
	if err != nil {
		return nil, err
	}

	props, err := resourceclient.VmProperties(vmInfo)
	if err != nil {
		return nil, err
	}

	var name string
	if vmInfo != nil && vmInfo.Config != nil {
		name = vmInfo.Config.Name
	}

	mqlVm, err := v.MotorRuntime.CreateResource("vsphere.vm",
		"moid", vm.Reference().Encode(),
		"name", name,
		"properties", props,
		"inventoryPath", vm.InventoryPath,
	)
	if err != nil {
		return nil, err
	}

	return mqlVm, nil
}

func (v *mqlEsxiCommand) id() (string, error) {
	return v.Command()
}

func (v *mqlEsxiCommand) init(args *resources.Args) (*resources.Args, EsxiCommand, error) {
	t := v.MotorRuntime.Motor.Provider
	vt, ok := t.(*provider.Provider)
	if !ok {
		return nil, nil, errors.New("esxi resource is only supported on vsphere transport")
	}

	if len(*args) > 2 {
		return args, nil, nil
	}

	// check if the command arg is provided
	commandRaw := (*args)["command"]
	if commandRaw == nil {
		return args, nil, nil
	}

	// check if the connection was initialized with a specific host
	identifier, err := vt.Identifier()
	if err != nil || !provider.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("could not determine inventoryPath from transport connection")
	}

	h, err := v.hostSystem(vt, identifier)
	if err != nil {
		return nil, nil, err
	}

	(*args)["inventoryPath"] = h.InventoryPath
	return args, nil, nil
}

func (v *mqlEsxiCommand) hostSystem(vt *provider.Provider, identifier string) (*object.HostSystem, error) {
	var h *object.HostSystem
	vClient := vt.Client()
	cl := resourceclient.New(vClient)

	// extract type and inventory
	moid, err := provider.ParseVsphereResourceID(identifier)
	if err != nil {
		return nil, err
	}

	if moid.Type != "HostSystem" {
		return nil, errors.New("esxi resource is not supported for vsphere type " + moid.Type)
	}

	h, err = cl.HostByMoid(moid)
	if err != nil {
		return nil, errors.New("could not find the esxi host via platform id: " + identifier)
	}

	return h, nil
}

func (v *mqlEsxiCommand) GetResult() ([]interface{}, error) {
	t := v.MotorRuntime.Motor.Provider
	_, ok := t.(*provider.Provider)
	if !ok {
		return nil, errors.New("esxi resource is not supported on this transport")
	}

	inventoryPath, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	esxiClient, err := esxiClient(v.MotorRuntime.Motor.Provider, inventoryPath)
	if err != nil {
		return nil, err
	}

	cmd, err := v.Command()
	if err != nil {
		return nil, err
	}

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

func (v *mqlVsphereVswitchStandard) id() (string, error) {
	return v.Name()
}

func (v *mqlVsphereVswitchStandard) esxiClient() (*resourceclient.Esxi, error) {
	c, ok := v.MqlResource().Cache.Load("_host_inventory_path")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}
	inventoryPath := c.Data.(string)
	return esxiClient(v.MotorRuntime.Motor.Provider, inventoryPath)
}

func (v *mqlVsphereVswitchStandard) GetFailoverPolicy() (map[string]interface{}, error) {
	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.VswitchStandardFailoverPolicy(name)
}

func (v *mqlVsphereVswitchStandard) GetSecurityPolicy() (map[string]interface{}, error) {
	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.VswitchStandardSecurityPolicy(name)
}

func (v *mqlVsphereVswitchStandard) GetShapingPolicy() (map[string]interface{}, error) {
	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.VswitchStandardShapingPolicy(name)
}

func (v *mqlVsphereVswitchStandard) GetUplinks() ([]interface{}, error) {
	raw, err := v.Properties()
	if err != nil {
		return nil, err
	}

	properties, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected properties structure for vsphere switch")
	}

	uplinksRaw := properties["Uplinks"]
	uplinkNames, ok := uplinksRaw.([]interface{})
	if !ok {
		return nil, errors.New("unexpected type for vsphere switch uplinks " + reflect.ValueOf(uplinksRaw).Type().Name())
	}

	// get the esxi.host parent resource
	c, ok := v.MqlResource().Cache.Load("_parent_resource")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}

	// get all host adapter
	host := c.Data.(VsphereHost)
	return findHostAdapter(host, uplinkNames)
}

func findHostAdapter(host VsphereHost, uplinkNames []interface{}) ([]interface{}, error) {
	adapters, err := host.Adapters()
	if err != nil {
		return nil, errors.New("cannot retrieve esxi host adapters")
	}

	// gather all adapters on that host so that we can find the adapter by name
	mqlUplinks := []interface{}{}
	for i := range adapters {
		adapter := adapters[i].(VsphereVmnic)
		for i := range uplinkNames {
			uplinkName := uplinkNames[i].(string)

			name, err := adapter.Name()
			if err != nil {
				return nil, errors.New("cannot retrieve esxi adapter name")
			}

			if name == uplinkName {
				mqlUplinks = append(mqlUplinks, adapter)
			}
		}
	}

	return mqlUplinks, nil
}

func (v *mqlVsphereVswitchDvs) id() (string, error) {
	return v.Name()
}

func (v *mqlVsphereVswitchDvs) GetUplinks() ([]interface{}, error) {
	raw, err := v.Properties()
	if err != nil {
		return nil, err
	}

	properties, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected properties structure for vsphere switch")
	}

	uplinksRaw := properties["Uplinks"]
	uplinkNames, ok := uplinksRaw.([]interface{})
	if !ok {
		return nil, errors.New("unexpected type for vsphere switch uplinks " + reflect.ValueOf(uplinksRaw).Type().Name())
	}

	// get the esxi.host parent resource
	c, ok := v.MqlResource().Cache.Load("_parent_resource")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}

	// get all host adapter
	host := c.Data.(VsphereHost)
	return findHostAdapter(host, uplinkNames)
}

package resources

import (
	"errors"
	"reflect"
	"time"

	"github.com/vmware/govmomi/vim25/mo"

	"github.com/vmware/govmomi/object"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/vsphere"
	"go.mondoo.io/mondoo/motor/providers"
	vsphere_transport "go.mondoo.io/mondoo/motor/providers/vsphere"
)

func getClientInstance(t providers.Transport) (*vsphere.Client, error) {
	vt, ok := t.(*vsphere_transport.Provider)
	if !ok {
		return nil, errors.New("vsphere resource is not supported on this transport")
	}

	cl := vsphere.New(vt.Client())
	return cl, nil
}

func esxiClient(t providers.Transport, path string) (*vsphere.Esxi, error) {
	client, err := getClientInstance(t)
	if err != nil {
		return nil, err
	}

	host, err := client.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	esxi := vsphere.NewEsxiClient(client.Client, path, host)
	return esxi, nil
}

func (v *lumiVsphereLicense) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereVmknic) id() (string, error) {
	return v.Name()
}

func (v *lumiEsxiVib) id() (string, error) {
	return v.Id()
}

func (v *lumiEsxiKernelmodule) id() (string, error) {
	return v.Name()
}

func (v *lumiEsxiService) id() (string, error) {
	return v.Key()
}

func (v *lumiEsxiTimezone) id() (string, error) {
	return v.Key()
}

func (v *lumiEsxiNtpconfig) id() (string, error) {
	return v.Id()
}

func (v *lumiVsphere) id() (string, error) {
	return "vsphere", nil
}

func (v *lumiVsphere) GetAbout() (map[string]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	return client.AboutInfo()
}

func (v *lumiVsphere) GetDatacenters() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// fetch datacenters
	dcs, err := client.ListDatacenters()
	if err != nil {
		return nil, err
	}

	// convert datacenter to lumi
	datacenters := make([]interface{}, len(dcs))
	for i, dc := range dcs {
		lumiDc, err := v.MotorRuntime.CreateResource("vsphere.datacenter",
			"moid", dc.Reference().Encode(),
			"name", dc.Name(),
			"inventoryPath", dc.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		datacenters[i] = lumiDc
	}

	return datacenters, nil
}

func (v *lumiVsphere) GetLicenses() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// fetch license
	lcs, err := client.ListLicenses()
	if err != nil {
		return nil, err
	}

	// convert licenses to lumi
	licenses := make([]interface{}, len(lcs))
	for i, l := range lcs {
		lumiLicense, err := v.MotorRuntime.CreateResource("vsphere.license",
			"name", l.Name,
			"total", int64(l.Total),
			"used", int64(l.Used),
		)
		if err != nil {
			return nil, err
		}

		licenses[i] = lumiLicense
	}

	return licenses, nil
}

func vsphereHosts(client *vsphere.Client, runtime *lumi.Runtime, vhosts []*object.HostSystem) ([]interface{}, error) {
	lumiHosts := make([]interface{}, len(vhosts))
	for i, h := range vhosts {

		hostInfo, err := vsphere.HostInfo(h)
		if err != nil {
			return nil, err
		}

		props, err := vsphere.HostProperties(hostInfo)
		if err != nil {
			return nil, err
		}

		var name string
		if hostInfo != nil {
			name = hostInfo.Name
		}

		lumiHost, err := runtime.CreateResource("vsphere.host",
			"moid", h.Reference().Encode(),
			"name", name,
			"properties", props,
			"inventoryPath", h.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		lumiHosts[i] = lumiHost
	}

	return lumiHosts, nil
}

func (v *lumiVsphereDatacenter) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereDatacenter) GetHosts() ([]interface{}, error) {
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

func (v *lumiVsphereDatacenter) GetClusters() ([]interface{}, error) {
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

	lumiClusters := make([]interface{}, len(vCluster))
	for i, c := range vCluster {

		props, err := client.ClusterProperties(c)
		if err != nil {
			return nil, err
		}

		lumiCluster, err := v.MotorRuntime.CreateResource("vsphere.cluster",
			"moid", c.Reference().Encode(),
			"name", c.Name(),
			"properties", props,
			"inventoryPath", c.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		lumiClusters[i] = lumiCluster
	}

	return lumiClusters, nil
}

func (v *lumiVsphereCluster) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereCluster) GetHosts() ([]interface{}, error) {
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

func (v *lumiVsphereHost) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereHost) init(args *lumi.Args) (*lumi.Args, VsphereHost, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	h, hostInfo, err := esxiHostProperties(v.MotorRuntime)
	if err != nil {
		return nil, nil, err
	}

	props, err := vsphere.HostProperties(hostInfo)
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

func (v *lumiVsphereHost) esxiClient() (*vsphere.Esxi, error) {
	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	return esxiClient(v.MotorRuntime.Motor.Provider, path)
}

func (v *lumiVsphereHost) GetStandardSwitch() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	vswitches, err := esxiClient.VswitchStandard()
	if err != nil {
		return nil, err
	}

	lumiVswitches := make([]interface{}, len(vswitches))
	for i, s := range vswitches {
		lumiVswitch, err := v.MotorRuntime.CreateResource("vsphere.vswitch.standard",
			"name", s["Name"],
			"properties", s,
		)
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		lumiVswitch.LumiResource().Cache.Store("_host_inventory_path", &lumi.CacheEntry{Data: esxiClient.InventoryPath})
		lumiVswitch.LumiResource().Cache.Store("_parent_resource", &lumi.CacheEntry{Data: v})

		lumiVswitches[i] = lumiVswitch
	}

	return lumiVswitches, nil
}

func (v *lumiVsphereHost) GetDistributedSwitch() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	vswitches, err := esxiClient.VswitchDvs()
	if err != nil {
		return nil, err
	}

	lumiVswitches := make([]interface{}, len(vswitches))
	for i, s := range vswitches {
		lumiVswitch, err := v.MotorRuntime.CreateResource("vsphere.vswitch.dvs",
			"name", s["Name"],
			"properties", s,
		)
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		lumiVswitch.LumiResource().Cache.Store("_host_inventory_path", &lumi.CacheEntry{Data: esxiClient.InventoryPath})
		lumiVswitch.LumiResource().Cache.Store("_parent_resource", &lumi.CacheEntry{Data: v})

		lumiVswitches[i] = lumiVswitch
	}

	return lumiVswitches, nil
}

func (v *lumiVsphereHost) GetAdapters() ([]interface{}, error) {
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

	lumiAdapters := make([]interface{}, len(adapters))
	for i, a := range adapters {
		nicName := a["Name"].(string)
		pParams := pauseParams[nicName]

		lumiAdapter, err := v.MotorRuntime.CreateResource("vsphere.vmnic",
			"name", nicName,
			"properties", a,
			"pauseParams", pParams,
		)
		if err != nil {
			return nil, err
		}

		// store host inventory path, so that sub resources can use that to quickly query more
		lumiAdapter.LumiResource().Cache.Store("_host_inventory_path", &lumi.CacheEntry{Data: esxiClient.InventoryPath})
		lumiAdapters[i] = lumiAdapter
	}

	return lumiAdapters, nil
}

func (v *lumiVsphereVmnic) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereVmnic) esxiClient() (*vsphere.Esxi, error) {
	c, ok := v.LumiResource().Cache.Load("_host_inventory_path")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}
	inventoryPath := c.Data.(string)
	return esxiClient(v.MotorRuntime.Motor.Provider, inventoryPath)
}

func (v *lumiVsphereVmnic) GetDetails() (map[string]interface{}, error) {
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

func (v *lumiVsphereHost) GetVmknics() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	vmknics, err := esxiClient.Vmknics()
	if err != nil {
		return nil, err
	}

	lumiVmknics := make([]interface{}, len(vmknics))
	for i := range vmknics {
		entry := vmknics[i]
		lumiVswitch, err := v.MotorRuntime.CreateResource("vsphere.vmknic",
			"name", entry.Properties["Name"],
			"properties", entry.Properties,
			"ipv4", entry.Ipv4,
			"ipv6", entry.Ipv6,
			"tags", sliceInterface(entry.Tags),
		)
		if err != nil {
			return nil, err
		}
		lumiVmknics[i] = lumiVswitch
	}

	return lumiVmknics, nil
}

func (v *lumiVsphereHost) GetPackages() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	vibs, err := esxiClient.Vibs()
	if err != nil {
		return nil, err
	}

	lumiPackages := make([]interface{}, len(vibs))
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

		lumiVib, err := v.MotorRuntime.CreateResource("esxi.vib",
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
		lumiPackages[i] = lumiVib
	}

	return lumiPackages, nil
}

func (v *lumiVsphereHost) GetAcceptanceLevel() (string, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return "", err
	}
	return esxiClient.SoftwareAcceptance()
}

func (v *lumiVsphereHost) GetKernelModules() ([]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	modules, err := esxiClient.KernelModules()
	if err != nil {
		return nil, err
	}

	lumiModules := make([]interface{}, len(modules))
	for i, m := range modules {
		lumiModule, err := v.MotorRuntime.CreateResource("esxi.kernelmodule",
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
		lumiModules[i] = lumiModule
	}

	return lumiModules, nil
}

func (v *lumiVsphereHost) GetAdvancedSettings() (map[string]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	host, err := client.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return vsphere.HostOptions(host)
}

func (v *lumiVsphereHost) GetServices() ([]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	host, err := client.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	services, err := vsphere.HostServices(host)
	if err != nil {
		return nil, err
	}
	lumiServices := make([]interface{}, len(services))
	for i, s := range services {
		lumiService, err := v.MotorRuntime.CreateResource("esxi.service",
			"key", s.Key,
			"label", s.Label,
			"required", s.Required,
			"uninstallable", s.Uninstallable,
			"running", s.Running,
			"ruleset", sliceInterface(s.Ruleset),
			"policy", s.Policy, // on, off, automatic
		)
		if err != nil {
			return nil, err
		}
		lumiServices[i] = lumiService
	}
	return lumiServices, nil
}

func sliceInterface(slice []string) []interface{} {
	res := make([]interface{}, len(slice))
	for i := range slice {
		res[i] = slice[i]
	}
	return res
}

func (v *lumiVsphereHost) GetTimezone() (interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	host, err := client.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	datetimeinfo, err := vsphere.HostDateTime(host)
	if err != nil {
		return nil, err
	}

	if datetimeinfo == nil {
		return nil, errors.New("vsphere does not return HostDateTimeSystem timezone information")
	}

	lumiTimezone, err := v.MotorRuntime.CreateResource("esxi.timezone",
		"key", datetimeinfo.TimeZone.Key,
		"name", datetimeinfo.TimeZone.Name,
		"offset", int64(datetimeinfo.TimeZone.GmtOffset),
		"description", datetimeinfo.TimeZone.Description,
	)
	if err != nil {
		return nil, err
	}

	return lumiTimezone, nil
}

func (v *lumiVsphereHost) GetNtp() (interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	host, err := client.HostByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	datetimeinfo, err := vsphere.HostDateTime(host)
	if err != nil {
		return nil, err
	}

	var server []interface{}
	var config []interface{}

	if datetimeinfo != nil && datetimeinfo.NtpConfig != nil {
		server = sliceInterface(datetimeinfo.NtpConfig.Server)
		config = sliceInterface(datetimeinfo.NtpConfig.ConfigFile)
	}

	lumiNtpConfig, err := v.MotorRuntime.CreateResource("esxi.ntpconfig",
		"id", "ntp "+host.InventoryPath,
		"server", server,
		"config", config,
	)
	if err != nil {
		return nil, err
	}

	return lumiNtpConfig, nil
}

func (v *lumiVsphereHost) GetSnmp() (map[string]interface{}, error) {
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.Snmp()
}

func (v *lumiVsphereDatacenter) GetVms() ([]interface{}, error) {
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

	vms, err := client.ListVirtualMachines(dc)
	if err != nil {
		return nil, err
	}

	lumiVms := make([]interface{}, len(vms))
	for i, vm := range vms {
		vmInfo, err := vsphere.VmInfo(vm)
		if err != nil {
			return nil, err
		}

		props, err := vsphere.VmProperties(vmInfo)
		if err != nil {
			return nil, err
		}

		var name string
		if vmInfo != nil && vmInfo.Config != nil {
			name = vmInfo.Config.Name
		}

		lumiVm, err := v.MotorRuntime.CreateResource("vsphere.vm",
			"moid", vm.Reference().Encode(),
			"name", name,
			"properties", props,
			"inventoryPath", vm.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		lumiVms[i] = lumiVm
	}

	return lumiVms, nil
}

func (v *lumiVsphereVm) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereVm) GetAdvancedSettings() (map[string]interface{}, error) {
	client, err := getClientInstance(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := v.InventoryPath()
	if err != nil {
		return nil, err
	}

	vm, err := client.VirtualMachineByInventoryPath(path)
	if err != nil {
		return nil, err
	}

	return vsphere.AdvancedSettings(vm)
}

func (v *lumiEsxi) id() (string, error) {
	return "esxi", nil
}

func esxiHostProperties(runtime *lumi.Runtime) (*object.HostSystem, *mo.HostSystem, error) {
	t := runtime.Motor.Provider
	vt, ok := t.(*vsphere_transport.Provider)
	if !ok {
		return nil, nil, errors.New("esxi resource is not supported on this transport")
	}

	var h *object.HostSystem
	vClient := vt.Client()
	cl := vsphere.New(vClient)
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
		if err != nil || !vsphere_transport.IsVsphereResourceID(identifier) {
			return nil, nil, errors.New("esxi resource is only supported for esxi connections or vsphere vm connections")
		}

		// extract type and inventory
		moid, err := vsphere_transport.ParseVsphereResourceID(identifier)
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
	hostInfo, err := vsphere.HostInfo(h)
	if err != nil {
		return nil, nil, err
	}

	return h, hostInfo, nil
}

// GetHost returns the information about the current ESXi host
// Deprecated: use vsphere.host resource instead
func (v *lumiEsxi) GetHost() (interface{}, error) {
	h, hostInfo, err := esxiHostProperties(v.MotorRuntime)
	if err != nil {
		return nil, err
	}

	props, err := vsphere.HostProperties(hostInfo)
	if err != nil {
		return nil, err
	}

	var name string
	if hostInfo != nil {
		name = hostInfo.Name
	}

	lumiHost, err := v.MotorRuntime.CreateResource("vsphere.host",
		"moid", h.Reference().Encode(),
		"name", name,
		"properties", props,
		"inventoryPath", h.InventoryPath,
	)
	if err != nil {
		return nil, err
	}
	return lumiHost, nil
}

func esxiVmProperties(runtime *lumi.Runtime) (*object.VirtualMachine, *mo.VirtualMachine, error) {
	t := runtime.Motor.Provider
	vt, ok := t.(*vsphere_transport.Provider)
	if !ok {
		return nil, nil, errors.New("esxi resource is not supported on this transport")
	}

	vClient := vt.Client()
	cl := vsphere.New(vClient)

	// check if the connection was initialized with a specific host
	identifier, err := vt.Identifier()
	if err != nil || !vsphere_transport.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("esxi resource is only supported for esxi connections or vsphere vm connections")
	}

	// extract type and inventory
	moid, err := vsphere_transport.ParseVsphereResourceID(identifier)
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

	vmInfo, err := vsphere.VmInfo(vm)
	if err != nil {
		return nil, nil, err
	}

	return vm, vmInfo, nil
}

func (v *lumiEsxi) GetVm() (interface{}, error) {
	vm, vmInfo, err := esxiVmProperties(v.MotorRuntime)
	if err != nil {
		return nil, err
	}

	props, err := vsphere.VmProperties(vmInfo)
	if err != nil {
		return nil, err
	}

	var name string
	if vmInfo != nil && vmInfo.Config != nil {
		name = vmInfo.Config.Name
	}

	lumiVm, err := v.MotorRuntime.CreateResource("vsphere.vm",
		"moid", vm.Reference().Encode(),
		"name", name,
		"properties", props,
		"inventoryPath", vm.InventoryPath,
	)
	if err != nil {
		return nil, err
	}

	return lumiVm, nil
}

func (v *lumiEsxiCommand) id() (string, error) {
	return v.Command()
}

func (v *lumiEsxiCommand) init(args *lumi.Args) (*lumi.Args, EsxiCommand, error) {
	t := v.MotorRuntime.Motor.Provider
	vt, ok := t.(*vsphere_transport.Provider)
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
	if err != nil || !vsphere_transport.IsVsphereResourceID(identifier) {
		return nil, nil, errors.New("could not determine inventoryPath from transport connection")
	}

	h, err := v.hostSystem(vt, identifier)
	if err != nil {
		return nil, nil, err
	}

	(*args)["inventoryPath"] = h.InventoryPath
	return args, nil, nil
}

func (v *lumiEsxiCommand) hostSystem(vt *vsphere_transport.Provider, identifier string) (*object.HostSystem, error) {
	var h *object.HostSystem
	vClient := vt.Client()
	cl := vsphere.New(vClient)

	// extract type and inventory
	moid, err := vsphere_transport.ParseVsphereResourceID(identifier)
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

func (v *lumiEsxiCommand) GetResult() ([]interface{}, error) {
	t := v.MotorRuntime.Motor.Provider
	_, ok := t.(*vsphere_transport.Provider)
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

func (v *lumiVsphereVswitchStandard) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereVswitchStandard) esxiClient() (*vsphere.Esxi, error) {
	c, ok := v.LumiResource().Cache.Load("_host_inventory_path")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}
	inventoryPath := c.Data.(string)
	return esxiClient(v.MotorRuntime.Motor.Provider, inventoryPath)
}

func (v *lumiVsphereVswitchStandard) GetFailoverPolicy() (map[string]interface{}, error) {
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

func (v *lumiVsphereVswitchStandard) GetSecurityPolicy() (map[string]interface{}, error) {
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

func (v *lumiVsphereVswitchStandard) GetShapingPolicy() (map[string]interface{}, error) {
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

func (v *lumiVsphereVswitchStandard) GetUplinks() ([]interface{}, error) {
	properties, err := v.Properties()
	if err != nil {
		return nil, err
	}

	uplinksRaw := properties["Uplinks"]
	uplinkNames, ok := uplinksRaw.([]interface{})
	if !ok {
		return nil, errors.New("unexpected type for vsphere switch uplinks " + reflect.ValueOf(uplinksRaw).Type().Name())
	}

	// get the esxi.host parent resource
	c, ok := v.LumiResource().Cache.Load("_parent_resource")
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
	lumiUplinks := []interface{}{}
	for i := range adapters {
		adapter := adapters[i].(VsphereVmnic)
		for i := range uplinkNames {
			uplinkName := uplinkNames[i].(string)

			name, err := adapter.Name()
			if err != nil {
				return nil, errors.New("cannot retrieve esxi adapter name")
			}

			if name == uplinkName {
				lumiUplinks = append(lumiUplinks, adapter)
			}
		}
	}

	return lumiUplinks, nil
}

func (v *lumiVsphereVswitchDvs) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereVswitchDvs) GetUplinks() ([]interface{}, error) {
	properties, err := v.Properties()
	if err != nil {
		return nil, err
	}

	uplinksRaw := properties["Uplinks"]
	uplinkNames, ok := uplinksRaw.([]interface{})
	if !ok {
		return nil, errors.New("unexpected type for vsphere switch uplinks " + reflect.ValueOf(uplinksRaw).Type().Name())
	}

	// get the esxi.host parent resource
	c, ok := v.LumiResource().Cache.Load("_parent_resource")
	if !ok {
		return nil, errors.New("cannot get esxi host inventory path")
	}

	// get all host adapter
	host := c.Data.(VsphereHost)
	return findHostAdapter(host, uplinkNames)
}

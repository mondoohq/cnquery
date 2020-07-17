package resources

import (
	"go.mondoo.io/mondoo/lumi/resources/vsphere"
)

func (v *lumiVsphereLicense) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereVm) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereVswitch) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereHost) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereVmnic) id() (string, error) {
	return v.Name()
}

func (v *lumiVsphereVmknic) id() (string, error) {
	return v.Name()
}

func (v *lumiEsxiVib) id() (string, error) {
	return v.Id()
}

func (v *lumiVsphere) id() (string, error) {
	return "vsphere", nil
}

var defaultCfg = &vsphere.Config{
	VSphereServerHost: "192.168.56.102",
	User:              "root",
	Password:          "password1!",
}

func (v *lumiVsphere) GetDatacenters() ([]interface{}, error) {
	client, err := vsphere.New(defaultCfg)
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
		lumiDc, err := v.Runtime.CreateResource("vsphere.datacenter",
			"moid", dc.Reference().Value,
			"name", dc.Name(),
			"inventorypath", dc.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		datacenters[i] = lumiDc
	}

	return datacenters, nil
}

func (v *lumiVsphere) GetLicenses() ([]interface{}, error) {
	client, err := vsphere.New(defaultCfg)
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
		lumiLicense, err := v.Runtime.CreateResource("vsphere.license",
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

func (v *lumiVsphereDatacenter) id() (string, error) {
	return v.Moid()
}

func (v *lumiVsphereDatacenter) GetHosts() ([]interface{}, error) {
	client, err := vsphere.New(defaultCfg)
	if err != nil {
		return nil, err
	}

	path, err := v.Inventorypath()
	if err != nil {
		return nil, err
	}

	dc, err := client.Datacenter(path)
	if err != nil {
		return nil, err
	}

	vhosts, err := client.ListHosts(dc)
	if err != nil {
		return nil, err
	}

	lumiHosts := make([]interface{}, len(vhosts))
	for i, h := range vhosts {

		props, err := client.HostProperties(h)
		if err != nil {
			return nil, err
		}

		lumiHost, err := v.Runtime.CreateResource("vsphere.host",
			"moid", h.Reference().Value,
			"name", h.Name(),
			"properties", props,
			"inventorypath", h.InventoryPath,
		)
		if err != nil {
			return nil, err
		}

		lumiHosts[i] = lumiHost
	}

	return lumiHosts, nil
}

func (v *lumiVsphereHost) esxiClient() (*vsphere.Esxi, error) {
	client, err := vsphere.New(defaultCfg)
	if err != nil {
		return nil, err
	}

	path, err := v.Inventorypath()
	if err != nil {
		return nil, err
	}

	host, err := client.Host(path)
	if err != nil {
		return nil, err
	}

	esxi := vsphere.NewEsxiClient(client.Client, host)
	return esxi, nil
}

func (v *lumiVsphereHost) GetStandardvswitch() ([]interface{}, error) {
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
		lumiVswitch, err := v.Runtime.CreateResource("vsphere.vswitch",
			"name", s["Name"],
			"properties", map[string]interface{}(s),
		)
		if err != nil {
			return nil, err
		}
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

	lumiAdapters := make([]interface{}, len(adapters))
	for i, a := range adapters {
		lumiVswitch, err := v.Runtime.CreateResource("vsphere.vmnic",
			"name", a["Name"],
			"properties", map[string]interface{}(a),
		)
		if err != nil {
			return nil, err
		}
		lumiAdapters[i] = lumiVswitch
	}

	return lumiAdapters, nil
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
	for i, n := range vmknics {
		lumiVswitch, err := v.Runtime.CreateResource("vsphere.vmnic",
			"name", n.Properties["Name"],
			"properties", map[string]interface{}(n.Properties),
		)
		if err != nil {
			return nil, err
		}
		lumiVmknics[i] = lumiVswitch
	}

	return lumiVmknics, nil

	return nil, nil
}

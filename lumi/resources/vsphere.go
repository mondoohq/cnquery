package resources

import (
	"go.mondoo.io/mondoo/lumi/resources/vsphere"
)

func (v *lumiVsphere) id() (string, error) {
	return "vsphere", nil
}

func (v *lumiVsphere) GetDatacenters() ([]interface{}, error) {
	cfg := &vsphere.Config{
		VSphereServerHost: "127.0.0.1:8989",
		User:              "user",
		Password:          "pass",
	}

	client, err := vsphere.New(cfg)
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
			"inventorypath", dc.InventoryPath()
		)
		if err != nil {
			return nil, err
		}

		datacenters[i] = lumiDc
	}

	return datacenters, nil
}

func (v *lumiVsphere) GetLicenses() ([]interface{}, error) {
	cfg := &vsphere.Config{
		VSphereServerHost: "127.0.0.1:8989",
		User:              "user",
		Password:          "pass",
	}

	client, err := vsphere.New(cfg)
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

func (v *lumiVsphereLicense) id() (string, error) {
	return v.Name()
}

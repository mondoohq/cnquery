package vsphere

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVSphere(t *testing.T) {
	cfg := &Config{
		VSphereServerHost: "127.0.0.1:8990",
		User:              "user",
		Password:          "pass",
	}

	client, err := New(cfg)
	require.NoError(t, err)

	// fetch datacenters
	dcs, err := client.ListDatacenters()
	require.NoError(t, err)
	assert.Equal(t, 1, len(dcs))

	// fetch license
	lcs, err := client.ListLicenses()
	require.NoError(t, err)
	assert.Equal(t, 1, len(lcs))

	for _, dc := range dcs {
		// list hosts
		hosts, err := client.ListHosts(dc)
		require.NoError(t, err)
		assert.Equal(t, 3, len(hosts))

		// list vms
		vms, err := client.ListVirtualMachines(dc)
		require.NoError(t, err)
		assert.Equal(t, 3, len(vms))
	}
}

func TestESXi(t *testing.T) {
	cfg := &Config{
		VSphereServerHost: "192.168.56.102",
		User:              "root",
		Password:          "password1!",
	}

	client, err := New(cfg)
	require.NoError(t, err)

	// fetch datacenters
	dcs, err := client.ListDatacenters()
	require.NoError(t, err)
	assert.Equal(t, 1, len(dcs))

	// // fetch license
	// lcs, err := client.ListLicenses()
	// require.NoError(t, err)
	// assert.Equal(t, 1, len(lcs))

	// list hosts
	for _, dc := range dcs {
		// list hosts
		hosts, err := client.ListHosts(dc)
		require.NoError(t, err)
		assert.Equal(t, 1, len(hosts))

		// test the first host
		e := Esxi{c: client.Client, host: hosts[0]}

		systemVersion, err := e.SystemVersion()
		require.NoError(t, err)
		assert.Equal(t, "VMware ESXi", systemVersion.Product)

		switches, err := e.VswitchStandard()
		require.NoError(t, err)
		assert.Equal(t, 2, len(switches))

		switches, err = e.VswitchDvs()
		require.NoError(t, err)
		assert.Equal(t, 0, len(switches))

		nics, err := e.Vmknics()
		require.NoError(t, err)
		assert.Equal(t, 1, len(nics))

		// list packages
		vibs, err := e.Vibs()
		require.NoError(t, err)
		assert.Equal(t, 136, len(vibs))

		// package acceptance level
		acceptance, err := e.SoftwareAcceptance()
		require.NoError(t, err)
		assert.Equal(t, "PartnerSupported", acceptance)

		// list kernel modules
		modules, err := e.KernelModules()
		require.NoError(t, err)
		assert.Equal(t, 98, len(modules))

		// list advanced settings
		settings, err := e.AdvancedSettings()
		require.NoError(t, err)
		// TODO: the ui displays 1043, we need to find the difference
		assert.Equal(t, 1069, len(settings))

		// all host options (overlaps with the advanced settings)
		settings, err = HostOptions(hosts[0])
		require.NoError(t, err)
		assert.Equal(t, 1045, len(settings))

		// get snmp settings
		snmpSettings, err := e.Snmp()
		require.NoError(t, err)
		assert.Equal(t, 10, len(snmpSettings))

		// list vms
		vms, err := client.ListVirtualMachines(dc)
		require.NoError(t, err)
		assert.Equal(t, 1, len(vms))

		vm := vms[0]
		vsettings, err := client.AdvancedSettings(vm)
		require.NoError(t, err)
		assert.Equal(t, 1, len(vsettings))
	}
}

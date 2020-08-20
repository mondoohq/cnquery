package vsphere

// TODO: include vsphere simulator to test the parser
// import (
// 	"context"
// 	"fmt"
// 	"net/url"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"github.com/vmware/govmomi"
// )

// func newClient(host string, user string, password string) (*Client, error) {
// 	u, err := url.Parse("https://" + host + "/sdk")
// 	if err != nil {
// 		return nil, err
// 	}
// 	u.User = url.UserPassword(user, password)

// 	ctx := context.Background()
// 	vc, err := govmomi.NewClient(ctx, u, true)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return New(vc), nil
// }

// func TestVSphere(t *testing.T) {
// 	client, err := newClient("127.0.0.1:8990", "user", "pass")
// 	require.NoError(t, err)

// 	// fetch datacenters
// 	dcs, err := client.ListDatacenters()
// 	require.NoError(t, err)
// 	assert.Equal(t, 1, len(dcs))

// 	// fetch license
// 	lcs, err := client.ListLicenses()
// 	require.NoError(t, err)
// 	assert.Equal(t, 1, len(lcs))

// 	for _, dc := range dcs {
// 		// list hosts
// 		hosts, err := client.ListHosts(dc)
// 		require.NoError(t, err)
// 		assert.Equal(t, 4, len(hosts))

// 		// list vms
// 		vms, err := client.ListVirtualMachines(dc)
// 		require.NoError(t, err)
// 		assert.Equal(t, 4, len(vms))

// 		// fetch cluster
// 		clusters, err := client.ListClusters(dc)
// 		require.NoError(t, err)
// 		assert.Equal(t, 1, len(clusters))

// 		cluster := clusters[0]
// 		props, err := client.ClusterProperties(cluster)
// 		require.NoError(t, err)
// 		fmt.Printf("%v", props)
// 	}
// }

// func TestESXi(t *testing.T) {
// 	client, err := newClient("192.168.56.104", "root", "password1!")
// 	require.NoError(t, err)

// 	// fetch datacenters
// 	dcs, err := client.ListDatacenters()
// 	require.NoError(t, err)
// 	assert.Equal(t, 1, len(dcs))

// 	// fetch cluster
// 	clusters, err := client.ListClusters(dcs[0])
// 	require.NoError(t, err)
// 	assert.Equal(t, 0, len(clusters))

// 	// // fetch license
// 	// lcs, err := client.ListLicenses()
// 	// require.NoError(t, err)
// 	// assert.Equal(t, 1, len(lcs))

// 	// list hosts
// 	// for _, dc := range dcs {
// 	// 	// list hosts
// 	// 	hosts, err := client.ListHosts(dc)
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 1, len(hosts))

// 	// 	// test the first host
// 	// 	e := Esxi{c: client.Client, host: hosts[0]}

// 	// 	switches, err := e.VswitchStandard()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 2, len(switches))

// 	// 	switches, err = e.VswitchDvs()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 0, len(switches))

// 	// 	nics, err := e.Vmknics()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 1, len(nics))

// 	// 	// list packages
// 	// 	vibs, err := e.Vibs()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 136, len(vibs))

// 	// 	// package acceptance level
// 	// 	acceptance, err := e.SoftwareAcceptance()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, "PartnerSupported", acceptance)

// 	// 	// list kernel modules
// 	// 	modules, err := e.KernelModules()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 98, len(modules))

// 	// 	// list advanced settings
// 	// 	settings, err := e.AdvancedSettings()
// 	// 	require.NoError(t, err)
// 	// 	// TODO: the ui displays 1043, we need to find the difference
// 	// 	assert.Equal(t, 1069, len(settings))

// 	// 	// all host options (overlaps with the advanced settings)
// 	// 	hostoptions, err := HostOptions(hosts[0])
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 1045, len(hostoptions))

// 	// 	// get snmp settings
// 	// 	snmpSettings, err := e.Snmp()
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 10, len(snmpSettings))

// 	// 	// list vms
// 	// 	vms, err := client.ListVirtualMachines(dc)
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 1, len(vms))

// 	// 	vm := vms[0]
// 	// 	vsettings, err := AdvancedSettings(vm)
// 	// 	require.NoError(t, err)
// 	// 	assert.Equal(t, 23, len(vsettings))
// 	// }
// }

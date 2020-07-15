package vsphere

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestVSphere(t *testing.T) {
	cfg := &Config{
		VSphereServerHost: "127.0.0.1:8989",
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

	// list hosts
	hosts, err := client.ListHosts()
	require.NoError(t, err)
	assert.Equal(t, 3, len(hosts))

	// // list vms
	// vms, err := client.ListVirtualMachines()
	// require.NoError(t, err)
	// assert.Equal(t, 4, len(vms))
}

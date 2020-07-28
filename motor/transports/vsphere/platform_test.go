package vsphere

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestVsphereResourceID(t *testing.T) {
	id := "//platformid.api.mondoo.app/runtime/vsphere/type/HostSystem/inventorypath/%2Fha-datacenter%2Fhost%2Flocalhost.%2Flocalhost.localdomain"
	ok := IsVsphereResourceID(id)
	assert.True(t, ok)

	id = "//platformid.api.mondoo.app/runtime/vsphere/type/VirtualMachine/inventorypath/%2Fha-datacenter%2Fvm%2Ftest"
	ok = IsVsphereResourceID(id)
	assert.True(t, ok)
}

func TestVsphereID(t *testing.T) {
	id := "//platformid.api.mondoo.app/runtime/vsphere/uuid/ha-host"
	ok := IsVsphereID(id)
	assert.True(t, ok)
}

func TestMrnParser(t *testing.T) {

	id := VsphereResourceID("HostSystem", "/ha-datacenter/host/localhost./localhost.localdomain")
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/vsphere/type/HostSystem/inventorypath/L2hhLWRhdGFjZW50ZXIvaG9zdC9sb2NhbGhvc3QuL2xvY2FsaG9zdC5sb2NhbGRvbWFpbg==", id)
	ok := IsVsphereResourceID(id)
	assert.True(t, ok)

	typ, inventory, err := ParseVsphereResourceID(id)

	require.NoError(t, err)
	assert.Equal(t, "HostSystem", typ)
	assert.Equal(t, "/ha-datacenter/host/localhost./localhost.localdomain", inventory)
}

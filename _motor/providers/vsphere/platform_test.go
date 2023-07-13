package vsphere

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/vim25/types"
)

func TestVsphereResourceID(t *testing.T) {
	// api
	id := "//platformid.api.mondoo.app/runtime/vsphere/instance/ha-host"
	ok := IsVsphereResourceID(id)
	assert.False(t, ok)

	// esxi host
	id = "//platformid.api.mondoo.app/runtime/vsphere/instance/ha-host/moid/HostSystem-ha-host"
	ok = IsVsphereResourceID(id)
	assert.True(t, ok)

	// vm
	id = "//platformid.api.mondoo.app/runtime/vsphere/instance/ha-host/moid/VirtualMachine-4"
	ok = IsVsphereResourceID(id)
	assert.True(t, ok)
}

func TestVsphereID(t *testing.T) {
	id := "//platformid.api.mondoo.app/runtime/vsphere/instance/ha-host"
	ok := IsVsphereID(id)
	assert.True(t, ok)
}

func TestMrnParser(t *testing.T) {

	id := VsphereResourceID("uuid", types.ManagedObjectReference{Type: "VirtualMachine", Value: "4"})
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/vsphere/instance/uuid/moid/VirtualMachine-4", id)
	ok := IsVsphereResourceID(id)
	assert.True(t, ok)

	moid, err := ParseVsphereResourceID(id)

	require.NoError(t, err)
	assert.Equal(t, "VirtualMachine", moid.Type)
	assert.Equal(t, "4", moid.Value)
}

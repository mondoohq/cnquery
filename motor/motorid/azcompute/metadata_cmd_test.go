package azcompute_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid/azcompute"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestCommandProviderLinux(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/metadata_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := azcompute.NewCommandInstanceMetadata(trans, p)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/azure/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/virtualMachines/examplevmname", mrn)
}

func TestCommandProviderWindows(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/metadata_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := azcompute.NewCommandInstanceMetadata(trans, p)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/azure/subscriptions/xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxx/resourceGroups/macikgo-test-may-23/providers/Microsoft.Compute/virtualMachines/examplevmname", mrn)
}

package gce_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/motorid/gce"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestCommandProviderLinux(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/metadata_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := gce.NewCommandInstanceMetadata(provider, p)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/6001244637815193808", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313", ident.ProjectID)
}

func TestCommandProviderWindows(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/metadata_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := gce.NewCommandInstanceMetadata(provider, p)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/5275377306317132843", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313", ident.ProjectID)
}

package gce_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid/gce"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestCommandProviderLinux(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/metadata_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := gce.NewCommandInstanceMetadata(trans, p)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/6001244637815193808", mrn)
}

func TestCommandProviderWindows(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/metadata_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := gce.NewCommandInstanceMetadata(trans, p)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/5275377306317132843", mrn)
}

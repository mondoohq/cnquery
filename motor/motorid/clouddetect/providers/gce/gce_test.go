package gce

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestDetectLinuxInstance(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/instance_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, relatedIdentifiers := Detect(provider, p)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/6001244637815193808", identifier)
	require.Len(t, relatedIdentifiers, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313", relatedIdentifiers[0])
}

func TestDetectWindowsInstance(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/instance_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, relatedIdentifiers := Detect(provider, p)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/5275377306317132843", identifier)
	require.Len(t, relatedIdentifiers, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/projects/mondoo-dev-262313", relatedIdentifiers[0])
}

func TestNoMatch(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/aws_instance.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, relatedIdentifiers := Detect(provider, p)

	assert.Empty(t, identifier)
	assert.Empty(t, relatedIdentifiers)
}

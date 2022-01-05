package gce

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestDetectLinuxInstance(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/instance_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier := Detect(trans, p)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/6001244637815193808", identifier)
}

func TestDetectWindowsInstance(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/instance_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier := Detect(trans, p)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/mondoo-dev-262313/zones/us-central1-a/instances/5275377306317132843", identifier)
}

func TestNoMatch(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/aws_instance.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier := Detect(trans, p)
	assert.Empty(t, identifier)
}

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestDetectInstance(t *testing.T) {
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/instance.toml"})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier := Detect(trans, p)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", identifier)
}

func TestDetectNotInstance(t *testing.T) {
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/notinstance.toml"})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier := Detect(trans, p)
	assert.Equal(t, "", identifier)
}

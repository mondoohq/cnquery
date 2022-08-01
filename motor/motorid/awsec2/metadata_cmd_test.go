package awsec2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestEC2RoleProviderInstanceIdentityUnix(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/instance-identity_document_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := awsec2.NewCommandInstanceMetadata(trans, p)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", mrn)
}

func TestEC2RoleProviderInstanceIdentityWindows(t *testing.T) {
	trans, err := mock.NewFromTomlFile("./testdata/instance-identity_document_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := awsec2.NewCommandInstanceMetadata(trans, p)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-east-1/instances/i-1234567890abcdef0", mrn)
}

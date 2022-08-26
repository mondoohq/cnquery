package awsec2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestEC2RoleProviderInstanceIdentityUnix(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/instance-identity_document_linux.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := awsec2.NewCommandInstanceMetadata(provider, p)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

func TestEC2RoleProviderInstanceIdentityWindows(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/instance-identity_document_windows.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := awsec2.NewCommandInstanceMetadata(provider, p)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-east-1/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

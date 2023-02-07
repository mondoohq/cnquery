package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestDetectInstance(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/instance.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, name, related := Detect(provider, p)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", identifier)
	assert.Equal(t, "ec2-name", name)
	require.Len(t, related, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", related[0])
}

func TestDetectInstanceArm(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/instancearm.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, name, related := Detect(provider, p)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", identifier)
	assert.Equal(t, "ec2-name", name)
	require.Len(t, related, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", related[0])
}

func TestDetectNotInstance(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/notinstance.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, name, related := Detect(provider, p)

	assert.Equal(t, "", identifier)
	assert.Equal(t, "", name)

	require.Len(t, related, 0)
}

func TestDetectConainer(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/container.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	identifier, name, related := Detect(provider, p)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/172746783610/regions/us-east-1/container/vjtest/f088b38d61ac45d6a946b5aebbe7197a/314e35e0-2d0a-4408-b37e-16063461d73a", identifier)
	assert.Equal(t, "fargate-app", name)
	require.Len(t, related, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/172746783610", related[0])
}

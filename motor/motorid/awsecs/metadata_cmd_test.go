package awsecsid

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestEC2RoleProviderInstanceIdentityUnix(t *testing.T) {
	provider, err := mock.NewFromTomlFile("./testdata/container-identity.toml")
	require.NoError(t, err)

	m, err := motor.New(provider)
	require.NoError(t, err)

	p, err := m.Platform()
	require.NoError(t, err)

	metadata := NewContainerMetadata(provider, p)
	ident, err := metadata.Identify()

	require.Nil(t, err)
	require.Equal(t, "fargate-app", ident.Name)
	require.Equal(t, "arn:aws:ecs:us-east-1:172746783610:container/vjtest/f088b38d61ac45d6a946b5aebbe7197a/314e35e0-2d0a-4408-b37e-16063461d73a", ident.ContainerArn)
	require.Equal(t, "f088b38d61ac45d6a946b5aebbe7197a-3681984407", ident.RuntimeID)
	require.Contains(t, ident.PlatformIds, "//platformid.api.mondoo.app/runtime/docker/containers/f088b38d61ac45d6a946b5aebbe7197a-3681984407")
	require.Contains(t, ident.PlatformIds, "//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/172746783610/regions/us-east-1/container/vjtest/f088b38d61ac45d6a946b5aebbe7197a/314e35e0-2d0a-4408-b37e-16063461d73a")
	require.Contains(t, ident.AccountPlatformID, "//platformid.api.mondoo.app/runtime/aws/accounts/172746783610")
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
)

func TestDetectInstance(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	identifier, name, related := Detect(conn, platform)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", identifier)
	assert.Equal(t, "ec2-name", name)
	require.Len(t, related, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", related[0])
}

func TestDetectInstanceArm(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instancearm.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	identifier, name, related := Detect(conn, platform)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", identifier)
	assert.Equal(t, "ec2-name", name)
	require.Len(t, related, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", related[0])
}

func TestDetectNotInstance(t *testing.T) {
	conn, err := mock.New(0, "./testdata/notinstance.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	identifier, name, related := Detect(conn, platform)

	assert.Equal(t, "", identifier)
	assert.Equal(t, "", name)

	require.Len(t, related, 0)
}

func TestDetectConainer(t *testing.T) {
	conn, err := mock.New(0, "./testdata/container.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	identifier, name, related := Detect(conn, platform)

	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/172746783610/regions/us-east-1/container/vjtest/f088b38d61ac45d6a946b5aebbe7197a/314e35e0-2d0a-4408-b37e-16063461d73a", identifier)
	assert.Equal(t, "fargate-app", name)
	require.Len(t, related, 1)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/172746783610", related[0])
}

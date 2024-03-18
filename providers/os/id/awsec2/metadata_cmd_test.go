// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
)

func TestEC2RoleProviderInstanceIdentityUnix(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance-identity_document_linux.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := NewCommandInstanceMetadata(conn, platform, nil)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "ec2-name", ident.InstanceName)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

func TestEC2RoleProviderInstanceIdentityUnixNoName(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance-identity_document_linux_no_tags.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := NewCommandInstanceMetadata(conn, platform, nil)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "i-1234567890abcdef0", ident.InstanceName)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

func TestEC2RoleProviderInstanceIdentityWindows(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance-identity_document_windows.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := NewCommandInstanceMetadata(conn, platform, nil)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "ec2-name-windows", ident.InstanceName)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-east-1/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

func TestEC2RoleProviderInstanceIdentityWindowsNoName(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance-identity_document_windows_no_tags.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	metadata := NewCommandInstanceMetadata(conn, platform, nil)
	ident, err := metadata.Identify()

	assert.Nil(t, err)
	assert.Equal(t, "i-1234567890abcdef0", ident.InstanceName)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-east-1/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

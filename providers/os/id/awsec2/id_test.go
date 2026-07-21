// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInstanceId(t *testing.T) {
	path := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/185972265011/regions/us-east-1/instances/i-07f67838ada5879af"
	id, err := ParseMondooInstanceID(path)
	require.NoError(t, err)
	assert.Equal(t, id.Account, "185972265011")
	assert.Equal(t, id.Region, "us-east-1")
	assert.Equal(t, id.Id, "i-07f67838ada5879af")

	path = "//platformid.api.mondoo.app/runtime/aws/ec2/v1/account/185972265011/regions/us-east-1/instances/i-07f67838ada5879af"
	_, err = ParseMondooInstanceID(path)
	assert.Error(t, err, "invalid aws ec2 instance id")

	path = "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/185972265011/regions/us/instances/i-07f67838ada5879af"
	_, err = ParseMondooInstanceID(path)
	assert.Error(t, err, "invalid aws ec2 instance id")
}

func TestParseSnapshotId(t *testing.T) {
	path := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/185972265011/regions/us-east-1/snapshots/snap-07f67838ada5879af"
	id, err := ParseMondooSnapshotID(path)
	require.NoError(t, err)
	assert.Equal(t, "185972265011", id.Account)
	assert.Equal(t, "us-east-1", id.Region)
	assert.Equal(t, "snap-07f67838ada5879af", id.Id)

	// an instance id must NOT validate as a snapshot id
	instancePath := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/185972265011/regions/us-east-1/instances/i-07f67838ada5879af"
	assert.False(t, IsValidMondooSnapshotId(instancePath))
	_, err = ParseMondooSnapshotID(instancePath)
	assert.Error(t, err)
}

func TestParseAccountId(t *testing.T) {
	path := "//platformid.api.mondoo.app/runtime/aws/accounts/185972265011"
	accountID, err := ParseMondooAccountID(path)
	require.NoError(t, err)
	require.Equal(t, "185972265011", accountID)
}

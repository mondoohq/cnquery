// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestParseInstanceId(t *testing.T) {
	path := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/185972265011/regions/us-east-1/instances/i-07f67838ada5879af"
	id, err := ParseMondooInstanceID(path)
	assert.NilError(t, err)
	assert.Equal(t, id.Account, "185972265011")
	assert.Equal(t, id.Region, "us-east-1")
	assert.Equal(t, id.Id, "i-07f67838ada5879af")

	path = "//platformid.api.mondoo.app/runtime/aws/ec2/v1/account/185972265011/regions/us-east-1/instances/i-07f67838ada5879af"
	id, err = ParseMondooInstanceID(path)
	assert.Error(t, err, "invalid aws ec2 instance id")

	path = "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/185972265011/regions/us/instances/i-07f67838ada5879af"
	id, err = ParseMondooInstanceID(path)
	assert.Error(t, err, "invalid aws ec2 instance id")
}

func TestParseAccountId(t *testing.T) {
	path := "//platformid.api.mondoo.app/runtime/aws/accounts/185972265011"
	accountID, err := ParseMondooAccountID(path)
	require.NoError(t, err)
	require.Equal(t, "185972265011", accountID)
}

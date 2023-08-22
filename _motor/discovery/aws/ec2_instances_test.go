// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"testing"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParseEc2PlatformId(t *testing.T) {
	uri := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa"

	p := ParseEc2PlatformID(uri)
	assert.NotNil(t, p)
	assert.Equal(t, "675173580680", p.Account)
	assert.Equal(t, "eu-west-1", p.Region)
	assert.Equal(t, "i-0e11b0762369fbefa", p.Instance)
}

func TestIsInstanceInGoodState(t *testing.T) {
	require.False(t, InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: aws_sdk.Int32(48)}))
	require.True(t, InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: aws_sdk.Int32(16)}))
	require.True(t, InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: aws_sdk.Int32(80)}))
	require.False(t, InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: aws_sdk.Int32(32)}))
	require.False(t, InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: aws_sdk.Int32(0)}))
}

func TestWhereFilter(t *testing.T) {
	filters := Ec2InstancesFilters{
		Regions: []string{"us-east-2"},
	}
	require.Equal(t, `region == "us-east-2"`, whereFilter(filters))
	filters = Ec2InstancesFilters{
		Regions: []string{"us-east-2", "us-east-1"},
	}
	require.Equal(t, `region == "us-east-2" || region == "us-east-1"`, whereFilter(filters))

	filters = Ec2InstancesFilters{
		InstanceIds: []string{"i-0899354"},
	}
	require.Equal(t, `instanceId == "i-0899354"`, whereFilter(filters))
	filters = Ec2InstancesFilters{
		InstanceIds: []string{"i-0899354", "i-98743087403"},
	}
	require.Equal(t, `instanceId == "i-0899354" || instanceId == "i-98743087403"`, whereFilter(filters))

	filters = Ec2InstancesFilters{
		Tags: map[string]string{"Name": "test"},
	}
	require.Equal(t, `tags["Name"] == "test"`, whereFilter(filters))

	filters = Ec2InstancesFilters{
		Tags: map[string]string{"Name": "test", "another": "test2"},
	}
	// go access map keys randomly so both are possible outputs here when building the filters
	expected := []string{`tags["Name"] == "test" || tags["another"] == "test2"`, `tags["another"] == "test2" || tags["Name"] == "test"`}
	require.Contains(t, expected, whereFilter(filters))

	filters = Ec2InstancesFilters{
		Regions: []string{"us-east-2"},
		Tags:    map[string]string{"Name": "test"},
	}
	require.Equal(t, `region == "us-east-2" && tags["Name"] == "test"`, whereFilter(filters))
}

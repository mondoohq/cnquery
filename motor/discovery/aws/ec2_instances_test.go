package aws_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.io/mondoo/motor/discovery/aws"
)

func TestParseParseEc2PlatformId(t *testing.T) {
	uri := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa"

	p := aws.ParseEc2PlatformID(uri)
	assert.NotNil(t, p)
	assert.Equal(t, "675173580680", p.Account)
	assert.Equal(t, "eu-west-1", p.Region)
	assert.Equal(t, "i-0e11b0762369fbefa", p.Instance)
}

func TestIsInstanceInGoodState(t *testing.T) {
	require.False(t, aws.InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: 48}))
	require.True(t, aws.InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: 16}))
	require.True(t, aws.InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: 80}))
	require.False(t, aws.InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: 32}))
	require.False(t, aws.InstanceIsInRunningOrStoppedState(&types.InstanceState{Code: 0}))
}

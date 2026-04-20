// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoscalingGroupServiceLinkedRole(t *testing.T) {
	t.Run("empty arn sets null state", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		result, err := g.serviceLinkedRole()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, g.ServiceLinkedRole.IsNull())
		assert.True(t, g.ServiceLinkedRole.IsSet())
	})
}

func TestAutoscalingGroupLaunchTemplate(t *testing.T) {
	t.Run("empty arn sets null state", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		result, err := g.launchTemplate()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, g.LaunchTemplate.IsNull())
		assert.True(t, g.LaunchTemplate.IsSet())
	})
}

func TestAutoscalingGroupSubnets(t *testing.T) {
	t.Run("no subnet ids returns empty slice", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		result, err := g.subnets()
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestPopulateAutoscalingGroupInternals(t *testing.T) {
	region := "us-east-1"
	accountID := "111122223333"

	t.Run("populates launch template arn from id", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		group := autoscalingtypes.AutoScalingGroup{
			LaunchTemplate: &autoscalingtypes.LaunchTemplateSpecification{
				LaunchTemplateId:   aws.String("lt-0123456789abcdef0"),
				LaunchTemplateName: aws.String("my-template"),
				Version:            aws.String("2"),
			},
		}
		populateAutoscalingGroupInternals(g, group, region, accountID)
		assert.Equal(t, "lt-0123456789abcdef0", g.cacheLaunchTemplateId)
		assert.Equal(t, "my-template", g.cacheLaunchTemplateName)
		assert.Equal(t, "arn:aws:ec2:us-east-1:111122223333:launch-template/lt-0123456789abcdef0", g.cacheLaunchTemplateArn)
	})

	t.Run("nil launch template leaves cache empty", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		populateAutoscalingGroupInternals(g, autoscalingtypes.AutoScalingGroup{}, region, accountID)
		assert.Empty(t, g.cacheLaunchTemplateArn)
	})

	t.Run("parses comma-separated VPCZoneIdentifier", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		group := autoscalingtypes.AutoScalingGroup{
			VPCZoneIdentifier: aws.String("subnet-aaa, subnet-bbb,subnet-ccc"),
		}
		populateAutoscalingGroupInternals(g, group, region, accountID)
		assert.Equal(t, []string{"subnet-aaa", "subnet-bbb", "subnet-ccc"}, g.cacheSubnetIds)
	})

	t.Run("empty VPCZoneIdentifier produces no subnet ids", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		group := autoscalingtypes.AutoScalingGroup{
			VPCZoneIdentifier: aws.String("  "),
		}
		populateAutoscalingGroupInternals(g, group, region, accountID)
		assert.Empty(t, g.cacheSubnetIds)
	})

	t.Run("nil VPCZoneIdentifier produces no subnet ids", func(t *testing.T) {
		g := &mqlAwsAutoscalingGroup{}
		populateAutoscalingGroupInternals(g, autoscalingtypes.AutoScalingGroup{}, region, accountID)
		assert.Empty(t, g.cacheSubnetIds)
	})
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func TestShouldExcludeInstance(t *testing.T) {
	instance := ec2types.Instance{
		InstanceId: aws.String("iid"),
		Tags: []ec2types.Tag{
			{
				Key:   aws.String("key-1"),
				Value: aws.String("val-1"),
			},
			{
				Key:   aws.String("key-2"),
				Value: aws.String("val-2"),
			},
		},
	}

	t.Run("should exclude instance by id", func(t *testing.T) {
		filters := connection.Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{
				"iid",
			},
			ExcludeTags: map[string]string{
				"key-3": "val3",
			},
		}
		require.True(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should exclude instance by matching tag", func(t *testing.T) {
		filters := connection.Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{
				"iid-2",
			},
			ExcludeTags: map[string]string{
				"key-2": "val2",
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should not exclude instance with only a matching tag key", func(t *testing.T) {
		filters := connection.Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{
				"iid-2",
			},
			ExcludeTags: map[string]string{
				"key-2": "val3",
				"key-3": "val3",
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should not exclude instance when instance id and tags don't match", func(t *testing.T) {
		filters := connection.Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{
				"iid-2",
			},
			ExcludeTags: map[string]string{
				"key-3": "val3",
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})
}

func TestDetermineApplicableRegions(t *testing.T) {
	t.Run("allow regions override initial region list", func(t *testing.T) {
		initialRegions := []string{"a", "b"}
		allowedRegions := []string{"b", "c"}
		excludedRegions := []string{}

		expected := []string{"b", "c"}
		actual := determineApplicableRegions(initialRegions, allowedRegions, excludedRegions)
		require.ElementsMatch(t, expected, actual)
	})

	t.Run("excluded regions work correctly", func(t *testing.T) {
		initialRegions := []string{"a", "b"}
		allowedRegions := []string{}
		excludedRegions := []string{"b"}

		expected := []string{"a"}
		actual := determineApplicableRegions(initialRegions, allowedRegions, excludedRegions)
		require.ElementsMatch(t, expected, actual)
	})

	t.Run("excluded regions not present in the initial slice are ignored", func(t *testing.T) {
		initialRegions := []string{"a", "b"}
		allowedRegions := []string{}
		excludedRegions := []string{"b", "c", "d", "e"}

		expected := []string{"a"}
		actual := determineApplicableRegions(initialRegions, allowedRegions, excludedRegions)
		require.ElementsMatch(t, expected, actual)
	})
}

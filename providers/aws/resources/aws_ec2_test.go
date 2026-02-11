// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
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
		filters := connection.DiscoveryFilters{
			General: connection.GeneralDiscoveryFilters{
				ExcludeTags: map[string]string{
					"key-3": "val3",
				},
			},
			Ec2: connection.Ec2DiscoveryFilters{
				ExcludeInstanceIds: []string{
					"iid",
				},
			},
		}
		require.True(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should exclude instance by matching tag", func(t *testing.T) {
		filters := connection.DiscoveryFilters{
			General: connection.GeneralDiscoveryFilters{
				ExcludeTags: map[string]string{
					"key-2": "val2",
				},
			},
			Ec2: connection.Ec2DiscoveryFilters{
				ExcludeInstanceIds: []string{
					"iid-2",
				},
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should not exclude instance with only a matching tag key", func(t *testing.T) {
		filters := connection.DiscoveryFilters{
			General: connection.GeneralDiscoveryFilters{
				ExcludeTags: map[string]string{
					"key-2": "val3",
					"key-3": "val3",
				},
			},
			Ec2: connection.Ec2DiscoveryFilters{
				ExcludeInstanceIds: []string{
					"iid-2",
				},
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should not exclude instance when instance id and tags don't match", func(t *testing.T) {
		filters := connection.DiscoveryFilters{
			General: connection.GeneralDiscoveryFilters{
				ExcludeTags: map[string]string{
					"key-3": "val3",
				},
			},
			Ec2: connection.Ec2DiscoveryFilters{
				ExcludeInstanceIds: []string{
					"iid-2",
				},
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should exclude instances with matching values for the same tag", func(t *testing.T) {
		filters := connection.DiscoveryFilters{
			General: connection.GeneralDiscoveryFilters{
				ExcludeTags: map[string]string{
					"key-1": "val-1,val-2,val-3",
				},
			},
			Ec2: connection.Ec2DiscoveryFilters{
				ExcludeInstanceIds: []string{},
			},
		}
		require.True(t, shouldExcludeInstance(instance, filters))
	})

	t.Run("should not exclude instances when no tag values match", func(t *testing.T) {
		filters := connection.DiscoveryFilters{
			General: connection.GeneralDiscoveryFilters{
				ExcludeTags: map[string]string{
					"key-1": "val-2,val-3",
					"key-2": "val-1,val-3",
					"key-3": "val-1,val-2",
				},
			},
			Ec2: connection.Ec2DiscoveryFilters{
				ExcludeInstanceIds: []string{},
			},
		}
		require.False(t, shouldExcludeInstance(instance, filters))
	})
}

func TestImdsSupport(t *testing.T) {
	t.Run("empty value returns none", func(t *testing.T) {
		assert.Equal(t, "none", imdsSupport(""))
	})

	t.Run("v2.0 value is preserved", func(t *testing.T) {
		assert.Equal(t, "v2.0", imdsSupport(ec2types.ImdsSupportValuesV20))
	})
}

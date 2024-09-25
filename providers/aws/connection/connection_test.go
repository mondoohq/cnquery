// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testParseOptsToFilters accepts a map which doesn't guarantee a deterministic iteration order. this means that slices
// in the parsed filters need to be compared individually ensuring their elements match regardless of their order.
func compareFilters(t *testing.T, expected, actual DiscoveryFilters) {
	require.ElementsMatch(t, expected.Ec2DiscoveryFilters.Regions, actual.Ec2DiscoveryFilters.Regions)
	require.ElementsMatch(t, expected.Ec2DiscoveryFilters.ExcludeRegions, actual.Ec2DiscoveryFilters.ExcludeRegions)

	require.ElementsMatch(t, expected.Ec2DiscoveryFilters.InstanceIds, actual.Ec2DiscoveryFilters.InstanceIds)
	require.ElementsMatch(t, expected.Ec2DiscoveryFilters.ExcludeInstanceIds, actual.Ec2DiscoveryFilters.ExcludeInstanceIds)

	require.Equal(t, expected.Ec2DiscoveryFilters.Tags, actual.Ec2DiscoveryFilters.Tags)
	require.Equal(t, expected.Ec2DiscoveryFilters.ExcludeTags, actual.Ec2DiscoveryFilters.ExcludeTags)

	require.Equal(t, expected.EcsDiscoveryFilters, actual.EcsDiscoveryFilters)

	require.ElementsMatch(t, expected.EcrDiscoveryFilters.Tags, actual.EcrDiscoveryFilters.Tags)

	require.ElementsMatch(t, expected.GeneralDiscoveryFilters.Regions, actual.GeneralDiscoveryFilters.Regions)
	require.Equal(t, expected.GeneralDiscoveryFilters.Tags, actual.GeneralDiscoveryFilters.Tags)
}

func TestParseOptsToFilters(t *testing.T) {
	t.Run("all opts are mapped to discovery filters correctly", func(t *testing.T) {
		opts := map[string]string{
			// Ec2DiscoveryFilters.Tags
			"ec2:tag:key1": "val1",
			"ec2:tag:key2": "val2",
			// Ec2DiscoveryFilters.ExcludeTags
			"exclude:ec2:tag:key1": "val1",
			"exclude:ec2:tag:key2": "val2",
			// Ec2DiscoveryFilters.Regions
			"ec2:region:us-east-1": "us-east-1",
			"ec2:region:us-west-1": "us-west-1",
			// Ec2DiscoveryFilters.ExcludeRegions
			"exclude:ec2:region:us-east-1": "us-east-1",
			"exclude:ec2:region:us-west-1": "us-west-1",
			// Ec2DiscoveryFilters.InstanceIds
			"ec2:instance-id:iid-1": "iid-1",
			"ec2:instance-id:iid-2": "iid-2",
			// Ec2DiscoveryFilters.ExcludeInstanceIds
			"exclude:ec2:instance-id:iid-1": "iid-1",
			"exclude:ec2:instance-id:iid-2": "iid-2",
			// GeneralDiscoveryFilters.Regions
			"all:region:us-east-1": "us-east-1",
			"all:region:us-west-1": "us-west-1",
			"region:eu-west-1":     "eu-west-1",
			// GeneralDiscoveryFilters.Tags
			"all:tag:key1": "val1",
			"all:tag:key2": "val2",
			// EcrDiscoveryFilters.Tags
			"ecr:tag:tag1": "tag1",
			"ecr:tag:tag2": "tag2",
			// EcsDiscoveryFilters
			"ecs:only-running-containers": "true",
			"ecs:discover-images":         "T",
			"ecs:discover-instances":      "false",
		}
		expected := DiscoveryFilters{
			Ec2DiscoveryFilters: Ec2DiscoveryFilters{
				Regions: []string{
					"us-east-1", "us-west-1",
				},
				ExcludeRegions: []string{
					"us-east-1", "us-west-1",
				},
				InstanceIds: []string{
					"iid-1", "iid-2",
				},
				ExcludeInstanceIds: []string{
					"iid-1", "iid-2",
				},
				Tags: map[string]string{
					"key1": "val1",
					"key2": "val2",
				},
				ExcludeTags: map[string]string{
					"key1": "val1",
					"key2": "val2",
				},
			},
			EcsDiscoveryFilters: EcsDiscoveryFilters{
				OnlyRunningContainers: true,
				DiscoverImages:        true,
				DiscoverInstances:     false,
			},
			EcrDiscoveryFilters: EcrDiscoveryFilters{Tags: []string{
				"tag1", "tag2",
			}},
			GeneralDiscoveryFilters: GeneralResourceDiscoveryFilters{
				Regions: []string{
					"us-east-1", "us-west-1", "eu-west-1",
				},
				Tags: map[string]string{
					"key1": "val1",
					"key2": "val2",
				},
			},
		}

		actual := parseOptsToFilters(opts)
		compareFilters(t, expected, actual)
	})

	t.Run("empty opts are mapped to discovery filters correctly", func(t *testing.T) {
		expected := DiscoveryFilters{
			Ec2DiscoveryFilters:     Ec2DiscoveryFilters{Tags: map[string]string{}, ExcludeTags: map[string]string{}},
			EcsDiscoveryFilters:     EcsDiscoveryFilters{},
			EcrDiscoveryFilters:     EcrDiscoveryFilters{Tags: []string{}},
			GeneralDiscoveryFilters: GeneralResourceDiscoveryFilters{Tags: map[string]string{}},
		}

		actual := parseOptsToFilters(map[string]string{})
		compareFilters(t, expected, actual)
	})
}

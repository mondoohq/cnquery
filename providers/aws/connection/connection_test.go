// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOptsToFilters(t *testing.T) {
	t.Run("all opts are mapped to discovery filters correctly", func(t *testing.T) {
		opts := map[string]string{
			// DiscoveryFilters.Regions
			"regions": "us-east-1,us-west-1,eu-west-1",
			// DiscoveryFilters.ExcludeRegions
			"exclude:regions": "us-east-2,us-west-2,eu-west-2",
			// Ec2DiscoveryFilters.InstanceIds
			"ec2:instance-ids": "iid-1,iid-2",
			// Ec2DiscoveryFilters.ExcludeInstanceIds
			"ec2:exclude:instance-ids": "iid-1,iid-2",
			// Ec2DiscoveryFilters.Tags
			"ec2:tag:key1": "val1",
			"ec2:tag:key2": "val2",
			// Ec2DiscoveryFilters.ExcludeTags
			"ec2:exclude:tag:key1": "val1,val2",
			"ec2:exclude:tag:key2": "val3",
			// EcrDiscoveryFilters.Tags
			"ecr:tags": "tag1,tag2",
			// EcrDiscoveryFilters.ExcludeTags
			"ecr:exclude:tags": "tag1,tag2",
			// EcsDiscoveryFilters
			"ecs:only-running-containers": "true",
			"ecs:discover-images":         "T",
			"ecs:discover-instances":      "false",
		}
		expected := DiscoveryFilters{
			DiscoveryFilters: GeneralDiscoveryFilters{
				Regions:        []string{"us-east-1", "us-west-1", "eu-west-1"},
				ExcludeRegions: []string{"us-east-2", "us-west-2", "eu-west-2"},
			},
			Ec2DiscoveryFilters: Ec2DiscoveryFilters{
				InstanceIds:        []string{"iid-1", "iid-2"},
				ExcludeInstanceIds: []string{"iid-1", "iid-2"},
				Tags: map[string]string{
					"key1": "val1",
					"key2": "val2",
				},
				ExcludeTags: map[string]string{
					"key1": "val1,val2",
					"key2": "val3",
				},
			},
			EcsDiscoveryFilters: EcsDiscoveryFilters{
				OnlyRunningContainers: true,
				DiscoverImages:        true,
				DiscoverInstances:     false,
			},
			EcrDiscoveryFilters: EcrDiscoveryFilters{
				Tags:        []string{"tag1", "tag2"},
				ExcludeTags: []string{"tag1", "tag2"},
			},
		}

		actual := parseOptsToFilters(opts)
		require.Equal(t, expected, actual)
	})

	t.Run("empty opts are mapped to discovery filters correctly", func(t *testing.T) {
		expected := EmptyDiscoveryFilters()
		actual := parseOptsToFilters(map[string]string{})
		require.Equal(t, expected, actual)
	})
}

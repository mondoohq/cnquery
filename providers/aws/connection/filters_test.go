// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/require"
)

func TestToServerSideEc2Filters(t *testing.T) {
	t.Run("correctly converts include and exclude tags to AWS SDK filters", func(t *testing.T) {
		filters := DiscoveryFilters{
			General: GeneralDiscoveryFilters{
				Tags: map[string]string{},
				// ignored
				ExcludeTags: map[string]string{
					"cost-center": "cc1,cc2",
				},
			},
		}

		sdkFilters := filters.General.ToServerSideEc2Filters()
		require.Empty(t, sdkFilters)
	})

	t.Run("correctly converts include and exclude tags to AWS SDK filters", func(t *testing.T) {
		filters := DiscoveryFilters{
			General: GeneralDiscoveryFilters{
				Tags: map[string]string{
					"env":  "prod,staging",
					"team": "alpha",
				},
				// ignored
				ExcludeTags: map[string]string{
					"cost-center": "cc1,cc2",
				},
			},
		}
		sdkFilters := filters.General.ToServerSideEc2Filters()
		expectedFilters := []ec2types.Filter{
			{
				Name:   aws.String("tag:env"),
				Values: []string{"prod", "staging"},
			},
			{
				Name:   aws.String("tag:team"),
				Values: []string{"alpha"},
			},
		}
		require.ElementsMatch(t, expectedFilters, sdkFilters)
	})
}

func TestMatchesIncludeTags(t *testing.T) {
	t.Run("no include tags matches", func(t *testing.T) {
		filters := GeneralDiscoveryFilters{
			Tags: map[string]string{},
		}
		resourceTags := map[string]string{
			"any-key": "any-value",
		}
		require.True(t, filters.MatchesIncludeTags(resourceTags))
	})

	t.Run("include tags do not match", func(t *testing.T) {
		filters := GeneralDiscoveryFilters{
			Tags: map[string]string{
				"tag1": "value1,value11",
				"tag2": "value2",
			},
		}
		resourceTags := map[string]string{
			"tag1": "value3",
		}
		require.False(t, filters.MatchesIncludeTags(resourceTags))
	})

	t.Run("include tags match", func(t *testing.T) {
		filters := GeneralDiscoveryFilters{
			Tags: map[string]string{
				"tag1": "value1,value11",
				"tag2": "value2",
			},
		}
		resourceTags := map[string]string{
			"tag1": "value11",
		}
		require.True(t, filters.MatchesIncludeTags(resourceTags))
	})
}

func TestMatchesExcludeTags(t *testing.T) {
	t.Run("no exclude tags does not match", func(t *testing.T) {
		filters := GeneralDiscoveryFilters{
			ExcludeTags: map[string]string{},
		}
		resourceTags := map[string]string{
			"any-key": "any-value",
		}
		require.False(t, filters.MatchesExcludeTags(resourceTags))
	})

	t.Run("exclude tags do not match", func(t *testing.T) {
		filters := GeneralDiscoveryFilters{
			ExcludeTags: map[string]string{
				"tag1": "value1,value11",
				"tag2": "value2",
			},
		}
		resourceTags := map[string]string{
			"tag1": "value3",
		}
		require.False(t, filters.MatchesExcludeTags(resourceTags))
	})

	t.Run("exclude tags match", func(t *testing.T) {
		filters := GeneralDiscoveryFilters{
			ExcludeTags: map[string]string{
				"tag1": "value1,value11",
				"tag2": "value2",
			},
		}
		resourceTags := map[string]string{
			"tag1": "value11",
		}
		require.True(t, filters.MatchesExcludeTags(resourceTags))
	})
}

func TestMatchesExcludeInstanceIds(t *testing.T) {
	t.Run("nil instanceId is not excluded", func(t *testing.T) {
		filters := Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{"i-1234567890abcdef0"},
		}
		require.False(t, filters.MatchesExcludeInstanceIds(nil))
	})

	t.Run("instanceId is not excluded", func(t *testing.T) {
		filters := Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{"i-1234567890abcdef0"},
		}
		require.False(t, filters.MatchesExcludeInstanceIds(aws.String("i-123456789notmatched")))
	})

	t.Run("instanceId is excluded", func(t *testing.T) {
		filters := Ec2DiscoveryFilters{
			ExcludeInstanceIds: []string{"i-1234567890abcdef0"},
		}
		require.True(t, filters.MatchesExcludeInstanceIds(aws.String("i-1234567890abcdef0")))
	})
}

func TestEcsMatchesOnlyRunningContainers(t *testing.T) {
	t.Run("OnlyRunningContainers is false, any container state matches", func(t *testing.T) {
		filters := EcsDiscoveryFilters{
			OnlyRunningContainers: false,
		}
		require.True(t, filters.MatchesOnlyRunningContainers("RUNNING"))
		require.True(t, filters.MatchesOnlyRunningContainers("STOPPED"))
		require.True(t, filters.MatchesOnlyRunningContainers("PENDING"))
	})

	t.Run("OnlyRunningContainers is true, only RUNNING container state matches", func(t *testing.T) {
		filters := EcsDiscoveryFilters{
			OnlyRunningContainers: true,
		}
		require.True(t, filters.MatchesOnlyRunningContainers("RUNNING"))
		require.False(t, filters.MatchesOnlyRunningContainers("STOPPED"))
		require.False(t, filters.MatchesOnlyRunningContainers("PENDING"))
	})
}

func TestEcrMatchesIncludeTags(t *testing.T) {
	t.Run("no include tags matches", func(t *testing.T) {
		filters := EcrDiscoveryFilters{
			Tags: []string{},
		}
		resourceTags := []string{"tag1", "tag2"}
		require.True(t, filters.MatchesIncludeTags(resourceTags))
	})

	t.Run("include tags do not match", func(t *testing.T) {
		filters := EcrDiscoveryFilters{
			Tags: []string{"tag3", "tag4"},
		}
		resourceTags := []string{"tag1", "tag2"}
		require.False(t, filters.MatchesIncludeTags(resourceTags))
	})

	t.Run("include tags match", func(t *testing.T) {
		filters := EcrDiscoveryFilters{
			Tags: []string{"tag1", "tag3"},
		}
		resourceTags := []string{"tag1", "tag2"}
		require.True(t, filters.MatchesIncludeTags(resourceTags))
	})
}

func TestEcrMatchesExcludeTags(t *testing.T) {
	t.Run("no exclude tags does not match", func(t *testing.T) {
		filters := EcrDiscoveryFilters{
			ExcludeTags: []string{},
		}
		resourceTags := []string{"tag1", "tag2"}
		require.False(t, filters.MatchesExcludeTags(resourceTags))
	})

	t.Run("exclude tags do not match", func(t *testing.T) {
		filters := EcrDiscoveryFilters{
			ExcludeTags: []string{"tag3", "tag4"},
		}
		resourceTags := []string{"tag1", "tag2"}
		require.False(t, filters.MatchesExcludeTags(resourceTags))
	})

	t.Run("exclude tags match", func(t *testing.T) {
		filters := EcrDiscoveryFilters{
			ExcludeTags: []string{"tag1", "tag3"},
		}
		resourceTags := []string{"tag1", "tag2"}
		require.True(t, filters.MatchesExcludeTags(resourceTags))
	})
}

func TestDiscoveryFiltersFromOpts(t *testing.T) {
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
			// GeneralDiscoveryFilters.Tags
			"tag:key1": "val1",
			"tag:key2": "val2",
			// GeneralDiscoveryFilters.ExcludeTags
			"exclude:tag:key1": "val1,val2",
			"exclude:tag:key2": "val3",
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
			General: GeneralDiscoveryFilters{
				Regions:        []string{"us-east-1", "us-west-1", "eu-west-1"},
				ExcludeRegions: []string{"us-east-2", "us-west-2", "eu-west-2"},
				Tags: map[string]string{
					"key1": "val1",
					"key2": "val2",
				},
				ExcludeTags: map[string]string{
					"key1": "val1,val2",
					"key2": "val3",
				},
			},
			Ec2: Ec2DiscoveryFilters{
				InstanceIds:        []string{"iid-1", "iid-2"},
				ExcludeInstanceIds: []string{"iid-1", "iid-2"},
			},
			Ecs: EcsDiscoveryFilters{
				OnlyRunningContainers: true,
				DiscoverImages:        true,
				DiscoverInstances:     false,
			},
			Ecr: EcrDiscoveryFilters{
				Tags:        []string{"tag1", "tag2"},
				ExcludeTags: []string{"tag1", "tag2"},
			},
		}

		actual := DiscoveryFiltersFromOpts(opts)
		require.Equal(t, expected, actual)
	})

	t.Run("empty opts are mapped to discovery filters correctly", func(t *testing.T) {
		expected := DiscoveryFilters{
			General: GeneralDiscoveryFilters{
				Regions:        []string{},
				ExcludeRegions: []string{},
				Tags:           map[string]string{},
				ExcludeTags:    map[string]string{},
			},
			Ec2: Ec2DiscoveryFilters{
				InstanceIds:        []string{},
				ExcludeInstanceIds: []string{},
			},
			Ecr: EcrDiscoveryFilters{
				Tags:        []string{},
				ExcludeTags: []string{},
			},
			Ecs: EcsDiscoveryFilters{},
		}
		actual := DiscoveryFiltersFromOpts(map[string]string{})
		require.Equal(t, expected, actual)
	})

	t.Run("nil opts are mapped to discovery filter correctly", func(t *testing.T) {
		expected := DiscoveryFilters{
			General: GeneralDiscoveryFilters{
				Regions:        []string{},
				ExcludeRegions: []string{},
				Tags:           map[string]string{},
				ExcludeTags:    map[string]string{},
			},
			Ec2: Ec2DiscoveryFilters{
				InstanceIds:        []string{},
				ExcludeInstanceIds: []string{},
			},
			Ecr: EcrDiscoveryFilters{
				Tags:        []string{},
				ExcludeTags: []string{},
			},
			Ecs: EcsDiscoveryFilters{},
		}
		actual := DiscoveryFiltersFromOpts(nil)
		require.Equal(t, expected, actual)
	})
}

func TestParseMapOpt(t *testing.T) {
	t.Run("parses map options correctly", func(t *testing.T) {
		opts := map[string]string{
			"tag:env":        "prod,staging",
			"tag:team":       "alpha",
			"exclude:tag:dc": "us-east,us-west",
		}
		result := parseMapOpt(opts, "tag:")
		expected := map[string]string{
			"env":  "prod,staging",
			"team": "alpha",
		}
		require.Equal(t, expected, result)
	})

	t.Run("returns empty map no matching keys", func(t *testing.T) {
		opts := map[string]string{
			"tag":            "some-value", // key is missing `:`
			"some-other-key": "some-value",
		}
		result := parseMapOpt(opts, "tag:")
		expected := map[string]string{}
		require.Equal(t, expected, result)
	})

	t.Run("returns empty map when there are no opts", func(t *testing.T) {
		opts := map[string]string{}
		result := parseMapOpt(opts, "tag:")
		expected := map[string]string{}
		require.Equal(t, expected, result)
	})
}

func TestParseCsvSliceOpt(t *testing.T) {
	t.Run("parses comma-separated values correctly", func(t *testing.T) {
		opts := map[string]string{
			"key": "value1,value2,value3",
		}
		result := parseCsvSliceOpt(opts, "key")
		expected := []string{"value1", "value2", "value3"}
		require.Equal(t, expected, result)
	})

	t.Run("returns empty slice for missing key", func(t *testing.T) {
		opts := map[string]string{}
		result := parseCsvSliceOpt(opts, "key")
		expected := []string{}
		require.Equal(t, expected, result)
	})

	t.Run("returns empty slice for empty value", func(t *testing.T) {
		opts := map[string]string{
			"key": "",
		}
		result := parseCsvSliceOpt(opts, "key")
		expected := []string{}
		require.Equal(t, expected, result)
	})
}

func TestParseBoolOpt(t *testing.T) {
	t.Run("parses true values correctly", func(t *testing.T) {
		trueValues := []string{"true", "TRUE", "t", "T", "1"}
		for _, val := range trueValues {
			result := parseBoolOpt(map[string]string{"key": val}, "key", false)
			require.True(t, result)
		}
	})

	t.Run("parses false values correctly", func(t *testing.T) {
		falseValues := []string{"false", "FALSE", "f", "F", "0"}
		for _, val := range falseValues {
			result := parseBoolOpt(map[string]string{"key": val}, "key", true)
			require.False(t, result)
		}
	})

	t.Run("returns default for missing key", func(t *testing.T) {
		result := parseBoolOpt(map[string]string{}, "key", true)
		require.True(t, result)

		result = parseBoolOpt(map[string]string{}, "key", false)
		require.False(t, result)
	})
}

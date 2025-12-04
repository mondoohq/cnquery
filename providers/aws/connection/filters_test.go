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
	t.Run("no exclude tags matches", func(t *testing.T) {
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

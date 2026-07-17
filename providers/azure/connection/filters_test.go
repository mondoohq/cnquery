// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoveryFiltersFromOpts(t *testing.T) {
	t.Run("nil opts yields empty filters", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(nil)
		assert.Empty(t, f.Subscriptions.Include)
		assert.Empty(t, f.Subscriptions.Exclude)
	})

	t.Run("include only", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscriptions": "sub-a,sub-b",
		})
		assert.Equal(t, []string{"sub-a", "sub-b"}, f.Subscriptions.Include)
		assert.Empty(t, f.Subscriptions.Exclude)
	})

	t.Run("exclude only", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscriptions-exclude": "sub-x",
		})
		assert.Empty(t, f.Subscriptions.Include)
		assert.Equal(t, []string{"sub-x"}, f.Subscriptions.Exclude)
	})

	t.Run("both include and exclude", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscriptions":         "sub-a",
			"subscriptions-exclude": "sub-x,sub-y",
		})
		assert.Equal(t, []string{"sub-a"}, f.Subscriptions.Include)
		assert.Equal(t, []string{"sub-x", "sub-y"}, f.Subscriptions.Exclude)
	})

	t.Run("empty values yield empty slices, not a single empty element", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscriptions":         "",
			"subscriptions-exclude": "",
		})
		assert.Empty(t, f.Subscriptions.Include)
		assert.Empty(t, f.Subscriptions.Exclude)
	})

	t.Run("propagate-subscription-tags parses to true", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"propagate-subscription-tags": "true",
		})
		assert.True(t, f.PropagateSubscriptionTags)
	})

	t.Run("propagate-subscription-tags defaults to false", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(nil)
		assert.False(t, f.PropagateSubscriptionTags)
		assert.Empty(t, f.SubscriptionTags)
	})

	t.Run("subscription-tag: entries parse into the override map", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscription-tag:env":  "prod",
			"subscription-tag:team": "payments",
		})
		assert.Equal(t, map[string]string{"env": "prod", "team": "payments"}, f.SubscriptionTags)
	})

	t.Run("subscription-tag: skips empty values", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscription-tag:env": "",
		})
		assert.Empty(t, f.SubscriptionTags)
	})
}

func TestSubscriptionsFilter_IsFilteredOut(t *testing.T) {
	tests := []struct {
		name    string
		filter  SubscriptionsFilter
		subID   string
		skipped bool
	}{
		{
			name:    "no filters keeps everything",
			filter:  SubscriptionsFilter{},
			subID:   "sub-a",
			skipped: false,
		},
		{
			name:    "include list keeps a listed subscription",
			filter:  SubscriptionsFilter{Include: []string{"sub-a", "sub-b"}},
			subID:   "sub-a",
			skipped: false,
		},
		{
			name:    "include list skips an unlisted subscription",
			filter:  SubscriptionsFilter{Include: []string{"sub-a", "sub-b"}},
			subID:   "sub-c",
			skipped: true,
		},
		{
			name:    "exclude list skips a listed subscription",
			filter:  SubscriptionsFilter{Exclude: []string{"sub-x"}},
			subID:   "sub-x",
			skipped: true,
		},
		{
			name:    "exclude list keeps an unlisted subscription",
			filter:  SubscriptionsFilter{Exclude: []string{"sub-x"}},
			subID:   "sub-a",
			skipped: false,
		},
		{
			name:    "include short-circuits: listed sub kept even if also excluded",
			filter:  SubscriptionsFilter{Include: []string{"sub-a"}, Exclude: []string{"sub-a"}},
			subID:   "sub-a",
			skipped: false,
		},
		{
			name:    "include short-circuits: unlisted sub skipped, exclude ignored",
			filter:  SubscriptionsFilter{Include: []string{"sub-a"}, Exclude: []string{"sub-b"}},
			subID:   "sub-b",
			skipped: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.skipped, tt.filter.IsFilteredOut(tt.subID))
		})
	}
}

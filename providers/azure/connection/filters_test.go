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
}

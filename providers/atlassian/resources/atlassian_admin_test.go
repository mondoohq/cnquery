// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Locks in the fix for the inverted err check that previously caused every
// last_active timestamp to be either nil (parse-success path) or a zero-value
// (parse-failure path).
func TestParseAtlassianTime(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, parseAtlassianTime(""))
	})

	t.Run("unparseable returns nil", func(t *testing.T) {
		assert.Nil(t, parseAtlassianTime("definitely-not-a-date"))
	})

	t.Run("RFC3339 with offset", func(t *testing.T) {
		got := parseAtlassianTime("2026-04-27T16:00:00-04:00")
		require.NotNil(t, got)
		want := time.Date(2026, 4, 27, 20, 0, 0, 0, time.UTC)
		assert.True(t, got.Equal(want), "expected %s, got %s", want, got)
	})

	t.Run("RFC3339 with Z", func(t *testing.T) {
		got := parseAtlassianTime("2026-04-27T20:00:00Z")
		require.NotNil(t, got)
		assert.True(t, got.Equal(time.Date(2026, 4, 27, 20, 0, 0, 0, time.UTC)))
	})
}

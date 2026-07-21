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

// TestExtractAtlassianCursor verifies we extract the raw cursor value from a
// page's Links.Next URL — the SDK list methods re-encode it as ?cursor=, so the
// full Next URL must not be passed back verbatim (which would break paging and
// silently truncate managed users / policies / domains to the first page).
func TestExtractAtlassianCursor(t *testing.T) {
	assert.Equal(t, "abc123",
		extractAtlassianCursor("https://api.atlassian.com/admin/v1/orgs/org-1/users?cursor=abc123"))
	assert.Equal(t, "x=y/z+w",
		extractAtlassianCursor("https://api.atlassian.com/admin/v1/orgs/org-1/domains?cursor=x%3Dy%2Fz%2Bw"))
	assert.Equal(t, "", extractAtlassianCursor(""))
	assert.Equal(t, "", extractAtlassianCursor("https://api.atlassian.com/admin/v1/orgs/org-1/policies"))
}

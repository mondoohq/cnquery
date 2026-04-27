// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraPriorityName(t *testing.T) {
	assert.Equal(t, "", jiraPriorityName(nil))
	assert.Equal(t, "High", jiraPriorityName(&models.PriorityScheme{Name: "High"}))
	assert.Equal(t, "", jiraPriorityName(&models.PriorityScheme{}))
}

func TestJiraResolutionName(t *testing.T) {
	assert.Equal(t, "", jiraResolutionName(nil))
	assert.Equal(t, "Won't Do", jiraResolutionName(&models.ResolutionScheme{Name: "Won't Do"}))
	assert.Equal(t, "", jiraResolutionName(&models.ResolutionScheme{}))
}

func TestJiraDateTime(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, jiraDateTime(nil))
	})

	t.Run("converts to UTC", func(t *testing.T) {
		ny, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		// 2026-04-27 12:00:00 EDT == 2026-04-27 16:00:00 UTC
		local := time.Date(2026, 4, 27, 12, 0, 0, 0, ny)
		scheme := models.DateTimeScheme(local)

		got := jiraDateTime(&scheme)
		require.NotNil(t, got)
		assert.Equal(t, time.UTC, got.Location())
		assert.Equal(t, 16, got.Hour())
		assert.True(t, got.Equal(local), "UTC conversion must preserve the instant")
	})
}

func TestJiraDate(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, jiraDate(nil))
	})

	t.Run("converts to UTC", func(t *testing.T) {
		ny, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		local := time.Date(2026, 4, 27, 12, 0, 0, 0, ny)
		scheme := models.DateScheme(local)

		got := jiraDate(&scheme)
		require.NotNil(t, got)
		assert.Equal(t, time.UTC, got.Location())
		assert.True(t, got.Equal(local), "UTC conversion must preserve the instant")
	})
}

func TestStringsToAny(t *testing.T) {
	assert.Equal(t, []any{}, stringsToAny(nil))
	assert.Equal(t, []any{}, stringsToAny([]string{}))
	assert.Equal(t, []any{"a", "b", "c"}, stringsToAny([]string{"a", "b", "c"}))
}

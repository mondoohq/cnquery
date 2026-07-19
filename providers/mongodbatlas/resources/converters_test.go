// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrSlice(t *testing.T) {
	assert.Equal(t, []any{}, strSlice(nil))
	assert.Equal(t, []any{}, strSlice([]string{}))
	assert.Equal(t, []any{"a", "b"}, strSlice([]string{"a", "b"}))
}

func TestTimePtr(t *testing.T) {
	assert.Nil(t, timePtr(time.Time{}), "zero time maps to nil")

	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	got := timePtr(now)
	require.NotNil(t, got)
	assert.Equal(t, now, *got)
}

func TestDictTime(t *testing.T) {
	assert.Nil(t, dictTime(time.Time{}), "zero time maps to nil so the dict value round-trips")

	ts := time.Date(2026, 7, 19, 10, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-07-19T10:30:00Z", dictTime(ts))
}

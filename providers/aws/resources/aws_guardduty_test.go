// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAwsTimestamp(t *testing.T) {
	t.Run("RFC3339 with Z suffix", func(t *testing.T) {
		ts := parseAwsTimestamp("2026-04-09T05:40:04Z")
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, time.April, ts.Month())
		assert.Equal(t, 9, ts.Day())
		assert.Equal(t, 5, ts.Hour())
		assert.Equal(t, 40, ts.Minute())
		assert.Equal(t, 4, ts.Second())
		assert.Equal(t, time.UTC, ts.Location())
	})

	t.Run("RFC3339 with timezone offset", func(t *testing.T) {
		ts := parseAwsTimestamp("2026-04-09T05:40:04+00:00")
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, 5, ts.Hour())
	})

	t.Run("timestamp without timezone (e.g. EC2 Verified Access)", func(t *testing.T) {
		ts := parseAwsTimestamp("2026-04-09T05:40:04")
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, time.April, ts.Month())
		assert.Equal(t, 9, ts.Day())
		assert.Equal(t, 5, ts.Hour())
		assert.Equal(t, 40, ts.Minute())
		assert.Equal(t, 4, ts.Second())
		assert.Equal(t, time.UTC, ts.Location())
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		ts := parseAwsTimestamp("")
		assert.Nil(t, ts)
	})

	t.Run("garbage string returns nil", func(t *testing.T) {
		ts := parseAwsTimestamp("not-a-timestamp")
		assert.Nil(t, ts)
	})

	t.Run("timestamp with non-RFC3339 timezone offset +0000 (e.g. Lambda layers)", func(t *testing.T) {
		ts := parseAwsTimestamp("2026-04-12T18:11:01.019+0000")
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, time.April, ts.Month())
		assert.Equal(t, 12, ts.Day())
		assert.Equal(t, 18, ts.Hour())
		assert.Equal(t, 11, ts.Minute())
		assert.Equal(t, 1, ts.Second())
	})

	t.Run("timestamp with non-RFC3339 negative timezone offset", func(t *testing.T) {
		ts := parseAwsTimestamp("2026-04-12T11:11:01.019-0700")
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, 11, ts.Hour())
	})

	t.Run("timestamp with milliseconds and Z suffix", func(t *testing.T) {
		ts := parseAwsTimestamp("2026-04-12T18:11:01.019Z")
		require.NotNil(t, ts)
		assert.Equal(t, 18, ts.Hour())
		assert.Equal(t, 11, ts.Minute())
	})
}

func TestParseAwsTimestampPtr(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		ts := parseAwsTimestampPtr(nil)
		assert.Nil(t, ts)
	})

	t.Run("valid RFC3339 string pointer", func(t *testing.T) {
		s := "2026-04-09T12:00:00Z"
		ts := parseAwsTimestampPtr(&s)
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, 12, ts.Hour())
	})

	t.Run("timestamp without timezone via pointer", func(t *testing.T) {
		s := "2026-04-10T05:51:33"
		ts := parseAwsTimestampPtr(&s)
		require.NotNil(t, ts)
		assert.Equal(t, 2026, ts.Year())
		assert.Equal(t, time.April, ts.Month())
		assert.Equal(t, 10, ts.Day())
		assert.Equal(t, 5, ts.Hour())
		assert.Equal(t, 51, ts.Minute())
		assert.Equal(t, 33, ts.Second())
		assert.Equal(t, time.UTC, ts.Location())
	})
}

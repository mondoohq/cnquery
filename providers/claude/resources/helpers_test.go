// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFamily(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"claude-opus-4-6", "opus"},
		{"claude-sonnet-5", "sonnet"},
		{"claude-haiku-4-5-20251001", "haiku"},
		{"claude-3-5-sonnet-20241022", "sonnet"},
		{"some-unknown-model", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFamily(tt.id))
		})
	}
}

func TestParseTime(t *testing.T) {
	// An empty string is a valid "not set" value and must yield the zero time
	// with no error, so a missing timestamp never fails the whole list call.
	zero, err := parseTime("")
	require.NoError(t, err)
	assert.True(t, zero.IsZero())

	got, err := parseTime("2026-01-02T15:04:05Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC), got.UTC())

	_, err = parseTime("not-a-timestamp")
	assert.Error(t, err)
}

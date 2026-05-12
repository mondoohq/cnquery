// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJamfTime_ValidRFC3339(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "UTC timestamp",
			input:    "2024-06-15T10:30:00Z",
			expected: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "with timezone offset",
			input:    "2024-01-01T00:00:00+05:00",
			expected: time.Date(2024, 1, 1, 0, 0, 0, 0, time.FixedZone("", 5*60*60)),
		},
		{
			name:     "with fractional seconds",
			input:    "2024-12-31T23:59:59.999Z",
			expected: time.Date(2024, 12, 31, 23, 59, 59, 999000000, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJamfTime(tt.input)
			require.NotNil(t, result)
			assert.True(t, tt.expected.Equal(*result), "expected %v, got %v", tt.expected, *result)
		})
	}
}

func TestParseJamfTime_EmptyString(t *testing.T) {
	result := parseJamfTime("")
	assert.Nil(t, result)
}

func TestParseJamfTime_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "date only", input: "2024-06-15"},
		{name: "garbage", input: "not-a-date"},
		{name: "unix timestamp", input: "1718450000"},
		{name: "US date format", input: "06/15/2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJamfTime(tt.input)
			assert.Nil(t, result)
		})
	}
}

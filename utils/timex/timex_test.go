// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package timex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		format      string
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:   "RFC3339 with format",
			input:  "2025-07-09T15:30:00Z",
			format: "rfc3339",
			want:   time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		{
			name:   "date only with format",
			input:  "2025-07-09",
			format: "date",
			want:   time.Date(2025, 7, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:   "time only with format",
			input:  "15:30:00",
			format: "time",
			want:   time.Date(0, 1, 1, 15, 30, 0, 0, time.UTC),
		},
		{
			name:   "kitchen format",
			input:  "3:30PM",
			format: "kitchen",
			want:   time.Date(0, 1, 1, 15, 30, 0, 0, time.UTC),
		},
		{
			name:        "valid format but invalid date",
			input:       "2025-13-45",
			format:      "date",
			wantErr:     true,
			errContains: "parsing time",
		},
		// Note: Invalid format names are currently ignored.
		// Auto-detection tests
		{
			name:  "auto-detect RFC3339",
			input: "2025-07-09T15:30:00Z",
			want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect DateTime",
			input: "2025-07-09 15:30:00",
			want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect DateOnly",
			input: "2025-07-09",
			want:  time.Date(2025, 7, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect TimeOnly",
			input: "15:30:00",
			want:  time.Date(0, 1, 1, 15, 30, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect RFC1123",
			input: "Wed, 09 Jul 2025 15:30:00 UTC",
			want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		// currently not supported because RFC1123 parses without error incorrectly
		// {
		// 	name:  "auto-detect RFC1123Z",
		// 	input: "Wed, 09 Jul 2025 15:30:00 +0000",
		// 	want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		// },
		{
			name:  "auto-detect ANSIC",
			input: "Wed Jul  9 15:30:00 2025",
			want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect RFC822",
			input: "09 Jul 25 15:30 UTC",
			want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		// currently not supported because RFC822 parses without error incorrectly
		// {
		// 	name:  "auto-detect RFC822Z",
		// 	input: "09 Jul 25 15:30 +0000",
		// 	want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		// },
		{
			name:  "auto-detect RFC850",
			input: "Wednesday, 09-Jul-25 15:30:00 UTC",
			want:  time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect Kitchen",
			input: "3:30PM",
			want:  time.Date(0, 1, 1, 15, 30, 0, 0, time.UTC),
		},
		{
			name:  "auto-detect Stamp",
			input: "Jul  9 15:30:00",
			want:  time.Date(0, 7, 9, 15, 30, 0, 0, time.UTC),
		},
		{
			name:        "auto-detect failure",
			input:       "not a date",
			wantErr:     true,
			errContains: "no supported date/time format found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, tt.format)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUTCTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "valid UTC time",
			input: "2026-03-17T14:30:00Z\n",
			want:  time.Date(2026, 3, 17, 14, 30, 0, 0, time.UTC),
		},
		{
			name:    "empty output",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not a date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUTCTime(strings.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTimezone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "IANA timezone from readlink",
			input: "America/New_York\n",
			want:  "America/New_York",
		},
		{
			name:  "timezone from /etc/timezone",
			input: "Europe/London\n",
			want:  "Europe/London",
		},
		{
			name:  "abbreviated timezone fallback",
			input: "EST\n",
			want:  "EST",
		},
		{
			name:  "empty defaults to UTC",
			input: "",
			want:  "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimezone(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

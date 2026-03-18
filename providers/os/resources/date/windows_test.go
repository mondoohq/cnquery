// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWindowsDate(t *testing.T) {
	w := &Windows{}

	tests := []struct {
		name     string
		input    string
		wantTZ   string
		wantYear int
		wantErr  bool
	}{
		{
			name:     "valid output",
			input:    `{"DateTime":"2026-03-17T14:30:00Z","Timezone":"America/New_York"}`,
			wantTZ:   "America/New_York",
			wantYear: 2026,
		},
		{
			name:     "UTC timezone",
			input:    `{"DateTime":"2026-03-17T14:30:00Z","Timezone":"UTC"}`,
			wantTZ:   "UTC",
			wantYear: 2026,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "invalid datetime",
			input:   `{"DateTime":"not-a-date","Timezone":"UTC"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := w.parse(strings.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTZ, got.Timezone)
			assert.Equal(t, tt.wantYear, got.Time.Year())
		})
	}
}

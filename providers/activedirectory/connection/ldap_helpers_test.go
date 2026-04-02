// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"
	"time"
)

func TestDecodeSID(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		want    string
		wantErr bool
	}{
		{
			name: "well-known Everyone SID S-1-1-0",
			raw:  []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			want: "S-1-1-0",
		},
		{
			name: "BUILTIN Administrators S-1-5-32-544",
			raw: []byte{
				0x01,                               // revision
				0x02,                               // sub-authority count
				0x00, 0x00, 0x00, 0x00, 0x00, 0x05, // authority = 5
				0x20, 0x00, 0x00, 0x00, // sub-auth 1 = 32 (BUILTIN)
				0x20, 0x02, 0x00, 0x00, // sub-auth 2 = 544 (Administrators)
			},
			want: "S-1-5-32-544",
		},
		{
			name: "domain SID S-1-5-21-3623811015-3361044348-30300820",
			raw: []byte{
				0x01,                               // revision
				0x04,                               // sub-authority count = 4
				0x00, 0x00, 0x00, 0x00, 0x00, 0x05, // authority = 5
				0x15, 0x00, 0x00, 0x00, // sub-auth 1 = 21
				0xC7, 0xF7, 0xFE, 0xD7, // sub-auth 2 = 3623811015 (little-endian)
				0x7C, 0x77, 0x55, 0xC8, // sub-auth 3 = 3361044348
				0x94, 0x5A, 0xCE, 0x01, // sub-auth 4 = 30300820
			},
			want: "S-1-5-21-3623811015-3361044348-30300820",
		},
		{
			name:    "too short",
			raw:     []byte{0x01, 0x02, 0x00},
			wantErr: true,
		},
		{
			name:    "truncated sub-authorities",
			raw:     []byte{0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x15, 0x00, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeSID(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeSID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("DecodeSID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileTimeToTime(t *testing.T) {
	tests := []struct {
		name string
		ft   int64
		want time.Time
	}{
		{
			name: "zero is zero time",
			ft:   0,
			want: time.Time{},
		},
		{
			name: "never expires is zero time",
			ft:   0x7FFFFFFFFFFFFFFF,
			want: time.Time{},
		},
		{
			// 2024-01-15 12:00:00 UTC
			// Unix timestamp: 1705320000
			// Windows FILETIME: (1705320000 * 10_000_000) + 116444736000000000
			//                 = 17053200000000000 + 116444736000000000
			//                 = 133497936000000000
			name: "2024-01-15 12:00:00 UTC",
			ft:   133497936000000000,
			want: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			// 2020-07-14 00:00:00 UTC => Unix 1594684800
			// FILETIME: (1594684800 * 10_000_000) + 116444736000000000 = 132391584000000000
			name: "2020-07-14 00:00:00 UTC",
			ft:   132391584000000000,
			want: time.Date(2020, 7, 14, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileTimeToTime(tt.ft)
			if !got.Equal(tt.want) {
				t.Errorf("FileTimeToTime(%d) = %v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestDurationToDays(t *testing.T) {
	tests := []struct {
		name string
		d    int64
		want int
	}{
		{"zero", 0, 0},
		// -90 days = -(90 * 86400 * 10_000_000) = -77760000000000
		{"-90 days", -77760000000000, 90},
		// -1 day = -(86400 * 10_000_000) = -864000000000
		{"-1 day", -864000000000, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DurationToDays(tt.d)
			if got != tt.want {
				t.Errorf("DurationToDays(%d) = %d, want %d", tt.d, got, tt.want)
			}
		})
	}
}

func TestDurationToMinutes(t *testing.T) {
	tests := []struct {
		name string
		d    int64
		want int
	}{
		{"zero (until admin unlocks)", 0, 0},
		// -30 minutes = -(30 * 60 * 10_000_000) = -18000000000
		{"-30 minutes", -18000000000, 30},
		// -1 minute = -(60 * 10_000_000) = -600000000
		{"-1 minute", -600000000, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DurationToMinutes(tt.d)
			if got != tt.want {
				t.Errorf("DurationToMinutes(%d) = %d, want %d", tt.d, got, tt.want)
			}
		})
	}
}

func TestFunctionalLevelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "2000"},
		{"2", "2003"},
		{"7", "2016"},
		{"99", "99"}, // unknown returns verbatim
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FunctionalLevelName(tt.input)
			if got != tt.want {
				t.Errorf("FunctionalLevelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

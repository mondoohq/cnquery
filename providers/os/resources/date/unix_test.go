// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package date

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
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

func TestExtractTZFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "standard Linux path",
			path: "/usr/share/zoneinfo/America/New_York",
			want: "America/New_York",
		},
		{
			name: "macOS path",
			path: "/var/db/timezone/zoneinfo/Europe/London",
			want: "Europe/London",
		},
		{
			name: "posix subdirectory path returns full relative",
			path: "/usr/share/zoneinfo/posix/Asia/Tokyo",
			want: "posix/Asia/Tokyo",
		},
		{
			name: "no zoneinfo marker",
			path: "/some/other/path",
			want: "",
		},
		{
			name: "zoneinfo at end with nothing after",
			path: "/usr/share/zoneinfo/",
			want: "",
		},
		{
			name: "localtime self-reference",
			path: "/usr/share/zoneinfo/localtime",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractTZFromPath(tt.path))
		})
	}
}

func TestTimezoneFromFS_EtcTimezone(t *testing.T) {
	// Simulate a Debian/Ubuntu system with /etc/timezone
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/etc/timezone", []byte("Europe/Berlin\n"), 0o644))

	tz, err := timezoneFromFS(fs)
	require.NoError(t, err)
	assert.Equal(t, "Europe/Berlin", tz)
}

func TestTimezoneFromFS_EtcTIMEZONE(t *testing.T) {
	// Simulate a Solaris/AIX system with /etc/TIMEZONE
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/etc/TIMEZONE", []byte("# timezone config\nTZ=US/Eastern\n"), 0o644))

	tz, err := timezoneFromFS(fs)
	require.NoError(t, err)
	assert.Equal(t, "US/Eastern", tz)
}

func TestTimezoneFromFS_LocaltimeBinaryMatch(t *testing.T) {
	// Simulate a Docker image where /etc/localtime is a copied TZif file
	fs := afero.NewMemMapFs()

	// Create a fake but valid TZif file
	tzifData := []byte("TZif" + strings.Repeat("\x00", 40) + "some timezone data here")

	// Write it as /etc/localtime (a regular file, not a symlink)
	require.NoError(t, afero.WriteFile(fs, "/etc/localtime", tzifData, 0o644))

	// Create a matching file in the zoneinfo tree
	require.NoError(t, afero.WriteFile(fs, "/usr/share/zoneinfo/America/Chicago", tzifData, 0o644))

	// Create a non-matching file to make sure we don't false-positive
	require.NoError(t, afero.WriteFile(fs, "/usr/share/zoneinfo/America/Denver", []byte("TZif"+strings.Repeat("\x00", 40)+"different data"), 0o644))

	tz, err := timezoneFromFS(fs)
	require.NoError(t, err)
	assert.Equal(t, "America/Chicago", tz)
}

func TestTimezoneFromFS_NoTimezoneInfo(t *testing.T) {
	// Empty filesystem - should fail
	fs := afero.NewMemMapFs()

	_, err := timezoneFromFS(fs)
	require.Error(t, err)
}

func TestTimezoneFromFS_SkipsPosixAndRight(t *testing.T) {
	fs := afero.NewMemMapFs()

	tzifData := []byte("TZif" + strings.Repeat("\x00", 40) + "unique tz data")
	require.NoError(t, afero.WriteFile(fs, "/etc/localtime", tzifData, 0o644))

	// Put matching file only under posix/ - should be skipped
	require.NoError(t, afero.WriteFile(fs, "/usr/share/zoneinfo/posix/America/Chicago", tzifData, 0o644))

	// Put real match under proper path
	require.NoError(t, afero.WriteFile(fs, "/usr/share/zoneinfo/America/Chicago", tzifData, 0o644))

	tz, err := timezoneFromFS(fs)
	require.NoError(t, err)
	assert.Equal(t, "America/Chicago", tz)
}

func TestTimezoneFromFS_InvalidLocaltime(t *testing.T) {
	fs := afero.NewMemMapFs()

	// /etc/localtime exists but isn't a valid TZif file
	require.NoError(t, afero.WriteFile(fs, "/etc/localtime", []byte("not a tzif file"), 0o644))

	_, err := timezoneFromFS(fs)
	require.Error(t, err)
}

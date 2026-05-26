// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHaproxyVersionRegex_RuntimeOutput covers the regex used against
// `haproxy -v` output. Both the legacy "HA-Proxy" and modern "HAProxy"
// banners must yield the same captured version body.
func TestHaproxyVersionRegex_RuntimeOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "modern 2.x banner",
			in:   "HAProxy version 2.8.24-eea79933d 2026/05/11 - https://haproxy.org/\nStatus: long-term supported branch",
			want: "2.8.24-eea79933d",
		},
		{
			name: "legacy 1.x banner",
			in:   "HA-Proxy version 1.8.31-1ppa1~jammy 2020/01/08 - https://haproxy.org/",
			want: "1.8.31-1ppa1~jammy",
		},
		{
			name: "no banner present",
			in:   "some unrelated text",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := reHaproxyVersion.FindStringSubmatch(tc.in)
			if tc.want == "" {
				assert.Nil(t, m)
				return
			}
			require.Len(t, m, 2)
			assert.Equal(t, tc.want, m[1])
		})
	}
}

// TestHaproxyEmbeddedVersionRegex guards against the regression where the
// previous binary-scan logic returned "2.5." from the deprecation message
// `parsing [...]: the '...' keyword is not supported any more since
// HAProxy version 2.5.`. The replacement anchors on the `, released
// YYYY/MM/DD` marker so only the real embedded version string matches.
func TestHaproxyEmbeddedVersionRegex(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "real embedded version literal",
			in:   "0\x00 version 2.8.24-eea79933d, released 2026/05/11\nGCC: (Debian 14.2.0-19) 14.2.0\n",
			want: "2.8.24-eea79933d",
		},
		{
			name: "deprecation message must NOT match",
			in:   "parsing [%s:%d]: the '%s' keyword is not supported any more since HAProxy version 2.5.\ndispatch",
			want: "",
		},
		{
			name: "printf format string must NOT match",
			in:   "HAProxy version %s %s - https://haproxy.org/\nStatus: long-term",
			want: "",
		},
		{
			name: "version without suffix",
			in:   "padding\x00 version 3.0.0, released 2026/01/15\nGCC:",
			want: "3.0.0",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := reHaproxyEmbeddedVersion.FindStringSubmatch(tc.in)
			if tc.want == "" {
				assert.Nil(t, m, "unexpected match: %v", m)
				return
			}
			require.Len(t, m, 2)
			assert.Equal(t, tc.want, m[1])
		})
	}
}

// TestScanHaproxyBinary_FullScan exercises the chunked-read path against
// an afero in-memory file that holds the embedded version literal beyond
// the first 64 KiB so the carry-over logic gets exercised.
func TestScanHaproxyBinary_FullScan(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Push the version literal past the first chunk boundary so the
	// rolling-overlap path is the one that finds it.
	padding := make([]byte, 80*1024)
	for i := range padding {
		padding[i] = 'x'
	}
	body := append(padding, []byte(" version 2.9.1, released 2026/03/01\nGCC:")...)
	require.NoError(t, afero.WriteFile(fs, "/usr/sbin/haproxy", body, 0o755))

	v := scanHaproxyBinary(&afero.Afero{Fs: fs}, "/usr/sbin/haproxy")
	assert.Equal(t, "2.9.1", v)

	// Non-existent path must return empty, not panic.
	assert.Equal(t, "", scanHaproxyBinary(&afero.Afero{Fs: fs}, "/nope"))

	// A binary that lacks the marker entirely returns empty (no false-
	// positive from a stray `version` token).
	require.NoError(t, afero.WriteFile(fs, "/no-marker", []byte("hello version foo bar"), 0o644))
	assert.Equal(t, "", scanHaproxyBinary(&afero.Afero{Fs: fs}, "/no-marker"))
}

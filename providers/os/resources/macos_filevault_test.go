// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// macos.filevault delegates command execution to the MQL runtime, but the
// interesting logic — picking the right line of `fdesetup status`, deciding
// what counts as "enabled", and turning `fdesetup list` into a slice of
// usernames — lives in pure helpers. The tests below drive those helpers
// with the exact strings macOS emits, so we can verify behavior without a
// live host.

func TestParseFdesetupStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "filevault on",
			raw:  "FileVault is On.\n",
			want: "FileVault is On.",
		},
		{
			name: "filevault off",
			raw:  "FileVault is Off.\n",
			want: "FileVault is Off.",
		},
		{
			name: "encryption in progress, multiple lines",
			// fdesetup follows the headline with a progress line during
			// active encryption; the headline alone is what we want.
			raw:  "Encryption in progress: Percent completed = 50.0\nFileVault master keychain appears to be installed.",
			want: "Encryption in progress: Percent completed = 50.0",
		},
		{
			name: "decryption in progress",
			raw:  "Decryption in progress: Percent completed = 25.0",
			want: "Decryption in progress: Percent completed = 25.0",
		},
		{
			name: "leading whitespace and blank line",
			// Defensive: don't return an empty first line.
			raw:  "\n   \nFileVault is On.\n",
			want: "FileVault is On.",
		},
		{
			name: "empty output",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace only",
			raw:  "   \n\t\n",
			want: "",
		},
		{
			name: "trailing whitespace trimmed",
			raw:  "FileVault is On.   \r\n",
			want: "FileVault is On.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFdesetupStatus(tt.raw))
		})
	}
}

func TestIsFilevaultEnabled(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "on",
			status: "FileVault is On.",
			want:   true,
		},
		{
			name:   "off",
			status: "FileVault is Off.",
			want:   false,
		},
		{
			name:   "encryption in progress counts as enabled",
			status: "Encryption in progress: Percent completed = 50.0",
			want:   true,
		},
		{
			// Decryption means the disk is on its way to plaintext — not enabled.
			name:   "decryption in progress is not enabled",
			status: "Decryption in progress: Percent completed = 75.0",
			want:   false,
		},
		{
			name:   "empty",
			status: "",
			want:   false,
		},
		{
			// Don't get fooled by substrings that don't actually announce status.
			name:   "unrelated text",
			status: "FileVault master keychain appears to be installed.",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isFilevaultEnabled(tt.status))
		})
	}
}

func TestParseFdesetupList(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []any
	}{
		{
			name: "single user",
			raw:  "alice,85632A00-1234-5678-ABCD-123456789ABC",
			want: []any{"alice"},
		},
		{
			name: "multiple users",
			raw:  "alice,85632A00-1234-5678-ABCD-123456789ABC\nbob,95632A00-1234-5678-ABCD-123456789ABC",
			want: []any{"alice", "bob"},
		},
		{
			name: "empty input yields empty slice (not nil)",
			raw:  "",
			want: []any{},
		},
		{
			name: "whitespace only yields empty slice",
			raw:  "   \n\n",
			want: []any{},
		},
		{
			name: "trailing newline ignored",
			raw:  "alice,UUID-A\nbob,UUID-B\n",
			want: []any{"alice", "bob"},
		},
		{
			// Belt-and-braces: missing UUID shouldn't crash; we still want the name.
			name: "line without comma falls back to whole line",
			raw:  "alice",
			want: []any{"alice"},
		},
		{
			name: "blank lines between users are skipped",
			raw:  "alice,UUID-A\n\nbob,UUID-B\n",
			want: []any{"alice", "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFdesetupList(tt.raw))
		})
	}
}

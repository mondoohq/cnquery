// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestParseFipsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"enabled", "1", true},
		{"enabled with newline", "1\n", true},
		{"enabled with whitespace", "  1  \n", true},
		{"disabled", "0", false},
		{"disabled with newline", "0\n", false},
		{"empty", "", false},
		{"whitespace only", "   \n", false},
		{"unexpected value", "2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseFipsEnabled(tt.content); got != tt.want {
				t.Errorf("parseFipsEnabled(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestNormalizeCryptoPolicy(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   string
	}{
		{"default with newline", "DEFAULT\n", "DEFAULT"},
		{"future no newline", "FUTURE", "FUTURE"},
		{"subpolicy kept as-is", "FIPS:OSPP", "FIPS:OSPP"},
		{"legacy with whitespace", "  LEGACY  \n", "LEGACY"},
		{"empty", "", ""},
		{"whitespace only", "   \n", ""},
		{"multiline takes first line", "DEFAULT\nsome warning\n", "DEFAULT"},
		{"fips", "FIPS\n", "FIPS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCryptoPolicy(tt.stdout); got != tt.want {
				t.Errorf("normalizeCryptoPolicy(%q) = %q, want %q", tt.stdout, got, tt.want)
			}
		})
	}
}

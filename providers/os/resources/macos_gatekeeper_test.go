// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// macos.gatekeeper drives `spctl --status` through the MQL runtime, but the
// status normalization and enabled-state decision are pure string ops. The
// tests below pin those helpers against the exact phrases macOS emits.

func TestParseSpctlStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "enabled with trailing newline",
			raw:  "assessments enabled\n",
			want: "assessments enabled",
		},
		{
			name: "disabled with trailing newline",
			raw:  "assessments disabled\n",
			want: "assessments disabled",
		},
		{
			name: "no trailing newline",
			raw:  "assessments enabled",
			want: "assessments enabled",
		},
		{
			name: "leading whitespace and CRLF",
			raw:  "   assessments enabled\r\n",
			want: "assessments enabled",
		},
		{
			name: "blank lines before status are skipped",
			raw:  "\n\nassessments enabled\n",
			want: "assessments enabled",
		},
		{
			name: "empty output",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace only",
			raw:  "\n  \t\n",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSpctlStatus(tt.raw))
		})
	}
}

func TestIsGatekeeperEnabled(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "enabled",
			status: "assessments enabled",
			want:   true,
		},
		{
			name:   "disabled",
			status: "assessments disabled",
			want:   false,
		},
		{
			name:   "empty",
			status: "",
			want:   false,
		},
		{
			// Make sure we don't match partial words — defends against future
			// spctl wording changes that might add prefixes or suffixes.
			name:   "extra suffix is not enabled",
			status: "assessments enabled (kernel)",
			want:   false,
		},
		{
			name:   "extra prefix is not enabled",
			status: "global: assessments enabled",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isGatekeeperEnabled(tt.status))
		})
	}
}

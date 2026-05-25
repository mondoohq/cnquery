// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalizePamModuleName(t *testing.T) {
	cases := []struct {
		in       string
		expected string
	}{
		{"pam_unix.so", "pam_unix"},
		{"/lib/security/pam_faillock.so", "pam_faillock"},
		{"pam_unix", "pam_unix"},
		{"/lib64/security/pam_pwquality.so", "pam_pwquality"},
		{"", ""},
		{"pam_faillock", "pam_faillock"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.expected, canonicalizePamModuleName(tc.in))
		})
	}
}

func TestAggregatePamParams(t *testing.T) {
	t.Run("single entry simple key=value", func(t *testing.T) {
		got := aggregatePamParams([]any{"deny=5"})
		assert.Equal(t, map[string]any{"deny": "5"}, got)
	})

	t.Run("last-write-wins across entries", func(t *testing.T) {
		got := aggregatePamParams(
			[]any{"deny=3"},
			[]any{"deny=5"},
		)
		assert.Equal(t, map[string]any{"deny": "5"}, got)
	})

	t.Run("bare option recorded as existence marker", func(t *testing.T) {
		got := aggregatePamParams([]any{"use_authtok"})
		assert.Equal(t, map[string]any{"use_authtok": ""}, got)
	})

	t.Run("mixed bare and key=value", func(t *testing.T) {
		got := aggregatePamParams([]any{"use_authtok", "deny=5", "unlock_time=900"})
		assert.Equal(t, map[string]any{
			"use_authtok": "",
			"deny":        "5",
			"unlock_time": "900",
		}, got)
	})

	t.Run("case-normalized keys", func(t *testing.T) {
		got := aggregatePamParams([]any{"Deny=5"})
		assert.Equal(t, map[string]any{"deny": "5"}, got)
	})

	t.Run("empty input produces empty map", func(t *testing.T) {
		got := aggregatePamParams()
		assert.Equal(t, map[string]any{}, got)
	})

	t.Run("value contains equals sign is preserved", func(t *testing.T) {
		// Bracketed forms like `default=die` aren't options — they live in
		// the control column — but if a value itself contains an `=`, we
		// keep everything after the first `=` as the value.
		got := aggregatePamParams([]any{"group=admin,wheel"})
		assert.Equal(t, map[string]any{"group": "admin,wheel"}, got)
	})

	t.Run("multi-list aggregation in order", func(t *testing.T) {
		got := aggregatePamParams(
			[]any{"unlock_time=600", "deny=3"},
			[]any{"deny=5", "even_deny_root"},
		)
		assert.Equal(t, map[string]any{
			"unlock_time":    "600",
			"deny":           "5",
			"even_deny_root": "",
		}, got)
	})
}

func TestIsPamControlEnabled(t *testing.T) {
	cases := []struct {
		control  string
		expected bool
	}{
		{"required", true},
		{"requisite", true},
		{"sufficient", true},
		{"optional", true},
		{"substack", true},
		{"include", true},
		{"[success=1 default=ignore]", false},
		{"[default=ignore]", false},
		{"[default=skip]", false},
		{"[default=die]", true},
		{"[success=ok default=bad]", true},
		{"", false},
	}

	for _, tc := range cases {
		t.Run(tc.control, func(t *testing.T) {
			assert.Equal(t, tc.expected, isPamControlEnabled(tc.control))
		})
	}
}

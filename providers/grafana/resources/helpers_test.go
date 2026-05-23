// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseGrafanaTime(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantNZ  bool // expect a non-zero result
		wantSub string
	}{
		{"empty", "", false, ""},
		{"invalid", "not-a-time", false, ""},
		{"grafana zero sentinel", "0001-01-01T00:00:00Z", false, ""},
		{"valid RFC3339 UTC", "2025-01-02T03:04:05Z", true, "2025-01-02"},
		{"valid RFC3339 offset", "2025-01-02T03:04:05+02:00", true, "2025-01-02"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGrafanaTime(tt.in)
			if tt.wantNZ {
				assert.False(t, got.IsZero(), "expected non-zero time, got zero")
				assert.Contains(t, got.UTC().Format(time.RFC3339), tt.wantSub)
			} else {
				assert.True(t, got.IsZero(), "expected zero time, got %v", got)
			}
		})
	}
}

func TestBoolFromAny(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want bool
	}{
		{"nil", nil, false},
		{"true bool", true, true},
		{"false bool", false, false},
		{"true string", "true", true},
		{"True string", "True", true},
		{"false string", "false", false},
		{"padded true", "  true  ", true},
		{"unparseable string", "yes", false},
		{"unrelated type", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, boolFromAny(tt.in))
		})
	}
}

func TestStringFromAny(t *testing.T) {
	assert.Equal(t, "", stringFromAny(nil))
	assert.Equal(t, "", stringFromAny(42))
	assert.Equal(t, "foo", stringFromAny("foo"))
}

func TestBoolFromJsonData(t *testing.T) {
	assert.False(t, boolFromJsonData(nil, "k"), "nil map → false")

	m := map[string]any{
		"yes":   true,
		"yesS":  "true",
		"no":    false,
		"empty": "",
	}
	assert.True(t, boolFromJsonData(m, "yes"))
	assert.True(t, boolFromJsonData(m, "yesS"))
	assert.False(t, boolFromJsonData(m, "no"))
	assert.False(t, boolFromJsonData(m, "empty"))
	assert.False(t, boolFromJsonData(m, "missing"), "missing key → false")
}

func TestContactPointURL(t *testing.T) {
	t.Run("nil settings", func(t *testing.T) {
		assert.Equal(t, "", contactPointURL("email", nil))
	})

	t.Run("url key wins", func(t *testing.T) {
		s := map[string]any{
			"url":        "https://hooks.example.com",
			"webhookUrl": "https://other.example.com",
		}
		assert.Equal(t, "https://hooks.example.com", contactPointURL("slack", s))
	})

	t.Run("falls back through candidates", func(t *testing.T) {
		// "url" missing → falls through to "webhookUrl"
		s := map[string]any{
			"webhookUrl": "https://wh.example.com",
		}
		assert.Equal(t, "https://wh.example.com", contactPointURL("webhook", s))
	})

	t.Run("non-string url ignored", func(t *testing.T) {
		s := map[string]any{
			"url": 42, // bogus type; must not panic, must return ""
		}
		assert.Equal(t, "", contactPointURL("slack", s))
	})

	t.Run("empty url ignored", func(t *testing.T) {
		s := map[string]any{
			"url":         "",
			"endpointUrl": "https://ep.example.com",
		}
		assert.Equal(t, "https://ep.example.com", contactPointURL("webhook", s))
	})
}

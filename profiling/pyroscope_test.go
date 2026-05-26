// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package profiling

import (
	"reflect"
	"testing"
)

func TestParseTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{"empty", "", map[string]string{}},
		{"single", "env=prod", map[string]string{"env": "prod"}},
		{"comma-separated", "env=prod,region=us-east-1", map[string]string{"env": "prod", "region": "us-east-1"}},
		{"semicolon-separated", "env=prod;region=us-east-1", map[string]string{"env": "prod", "region": "us-east-1"}},
		{"whitespace-trimmed", " env = prod , region = us-east-1 ", map[string]string{"env": "prod", "region": "us-east-1"}},
		{"value-with-equals", "header=Authorization=Bearer abc", map[string]string{"header": "Authorization=Bearer abc"}},
		{"empty-value", "env=", map[string]string{"env": ""}},
		{"skip-malformed", "env=prod,nokey,region=us-east-1", map[string]string{"env": "prod", "region": "us-east-1"}},
		{"trailing-separator", "env=prod,", map[string]string{"env": "prod"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTags(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestStartDisabledByDefault(t *testing.T) {
	t.Setenv(EnvEnabled, "")
	stopper, err := Start("test-service", nil)
	if err != nil {
		t.Fatalf("Start with profiling disabled returned error: %v", err)
	}
	if stopper == nil {
		t.Fatal("Start returned nil stopper; expected no-op")
	}
	if err := stopper.Stop(); err != nil {
		t.Errorf("Stop() on no-op stopper returned error: %v", err)
	}
}

func TestStartEnabledWithoutServerAddressErrors(t *testing.T) {
	t.Setenv(EnvEnabled, "true")
	t.Setenv(EnvServerAddress, "")
	stopper, err := Start("test-service", nil)
	if err == nil {
		t.Fatal("expected error when enabled without server address")
	}
	if stopper == nil {
		t.Fatal("Start should always return a non-nil stopper, even on error")
	}
}

func TestIsEnabled(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"no":    false,
		"1":     true,
		"true":  true,
		"TRUE":  true,
		"yes":   true,
		"on":    true,
	}
	for v, want := range cases {
		t.Setenv(EnvEnabled, v)
		if got := isEnabled(); got != want {
			t.Errorf("isEnabled() with %s=%q = %v, want %v", EnvEnabled, v, got, want)
		}
	}
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexTimeUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantMs  int64
	}{
		{name: "epoch millis", input: `1600000000000`, wantMs: 1600000000000},
		{name: "rfc3339 string", input: `"2021-01-01T00:00:00Z"`, wantMs: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()},
		{name: "json null", input: `null`, wantNil: true},
		{name: "empty string", input: `""`, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft flexTime
			if err := json.Unmarshal([]byte(tt.input), &ft); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got := ft.Time()
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected time, got nil")
			}
			if got.UnixMilli() != tt.wantMs {
				t.Fatalf("expected %d ms, got %d ms", tt.wantMs, got.UnixMilli())
			}
		})
	}
}

func TestParseTargets(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "array", input: `["production","preview"]`, want: []string{"production", "preview"}},
		{name: "single string", input: `"production"`, want: []string{"production"}},
		{name: "null", input: `null`, want: nil},
		{name: "empty string", input: `""`, want: nil},
		{name: "empty array", input: `[]`, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTargets(json.RawMessage(tt.input))
			if len(got) != len(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("index %d: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestFirewallRuleAction(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "plain string", input: "deny", want: "deny"},
		{name: "nested mitigate", input: map[string]any{"mitigate": map[string]any{"action": "deny"}}, want: "deny"},
		{name: "flat action", input: map[string]any{"action": "log"}, want: "log"},
		{name: "unknown shape", input: map[string]any{"foo": "bar"}, want: ""},
		{name: "nil", input: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firewallRuleAction(tt.input); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

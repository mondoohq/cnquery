// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestNicsHavePublicIp(t *testing.T) {
	cases := []struct {
		name string
		nics []any
		want bool
	}{
		{"has public ip", []any{map[string]any{"publicIp": "203.0.113.5"}}, true},
		{"second nic has ip", []any{map[string]any{"publicIp": ""}, map[string]any{"publicIp": "203.0.113.5"}}, true},
		{"empty public ip", []any{map[string]any{"publicIp": ""}}, false},
		{"no public ip key", []any{map[string]any{"nicId": "x"}}, false},
		{"no nics", []any{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := nicsHavePublicIp(c.nics); got != c.want {
				t.Errorf("nicsHavePublicIp(%v) = %v, want %v", c.nics, got, c.want)
			}
		})
	}
}

func TestSecurityRuleOpenToInternet(t *testing.T) {
	cases := []struct {
		direction string
		ipRange   string
		want      bool
	}{
		{"ingress", "0.0.0.0/0", true},
		{"ingress", "::/0", true},
		{"INGRESS", "0.0.0.0/0", true},
		{"ingress", "203.0.113.0/24", false},
		{"ingress", "", false},
		{"egress", "0.0.0.0/0", false},
	}
	for _, c := range cases {
		if got := securityRuleOpenToInternet(c.direction, c.ipRange); got != c.want {
			t.Errorf("securityRuleOpenToInternet(%q, %q) = %v, want %v", c.direction, c.ipRange, got, c.want)
		}
	}
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestSourceIsAnywhere(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"empty is any", "", true},
		{"whitespace is any", "   ", true},
		{"ipv4 default route", "0.0.0.0/0", true},
		{"ipv6 default route", "::/0", true},
		{"bare all-zero ipv4", "0.0.0.0", true},
		{"bare all-zero ipv6", "::", true},
		{"padded default route", " 0.0.0.0/0 ", true},
		{"list with any", "10.0.0.0/8,0.0.0.0/0", true},
		{"specific host", "203.0.113.5", false},
		{"specific cidr", "10.0.0.0/8", false},
		{"ipset reference", "+myset", false},
		{"alias reference", "dc/myalias", false},
		{"negated source", "!0.0.0.0/0", false},
		{"non-zero /0 host", "192.168.0.0/0", false},
		{"list without any", "10.0.0.0/8,192.168.0.0/16", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sourceIsAnywhere(tt.source); got != tt.want {
				t.Errorf("sourceIsAnywhere(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestFirewallRuleAllowsPublicIngress(t *testing.T) {
	tests := []struct {
		name     string
		ruleType string
		action   string
		source   string
		enable   bool
		want     bool
	}{
		{"inbound accept any", "in", "ACCEPT", "0.0.0.0/0", true, true},
		{"inbound accept empty source", "in", "ACCEPT", "", true, true},
		{"empty type defaults inbound", "", "ACCEPT", "0.0.0.0/0", true, true},
		{"lowercase action", "in", "accept", "::/0", true, true},
		{"disabled rule", "in", "ACCEPT", "0.0.0.0/0", false, false},
		{"outbound rule", "out", "ACCEPT", "0.0.0.0/0", true, false},
		{"group rule", "group", "ACCEPT", "0.0.0.0/0", true, false},
		{"drop action", "in", "DROP", "0.0.0.0/0", true, false},
		{"reject action", "in", "REJECT", "0.0.0.0/0", true, false},
		{"scoped source", "in", "ACCEPT", "10.0.0.0/8", true, false},
		{"ipset source", "in", "ACCEPT", "+trusted", true, false},
		{"padded inbound accept", " in ", " ACCEPT ", "0.0.0.0/0", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firewallRuleAllowsPublicIngress(tt.ruleType, tt.action, tt.source, tt.enable)
			if got != tt.want {
				t.Errorf("firewallRuleAllowsPublicIngress(%q,%q,%q,%v) = %v, want %v",
					tt.ruleType, tt.action, tt.source, tt.enable, got, tt.want)
			}
		})
	}
}

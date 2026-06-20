// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAnyAddress(t *testing.T) {
	cases := []struct {
		cidr string
		want bool
	}{
		{"0.0.0.0/0", true},
		{"::/0", true},
		{"0.0.0.0", true},
		{"::", true},
		{" 0.0.0.0/0 ", true}, // surrounding whitespace tolerated
		{"10.0.0.0/8", false},
		{"203.0.113.5/32", false},
		{"2001:db8::/32", false},
		{"", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, isAnyAddress(c.cidr), "isAnyAddress(%q)", c.cidr)
	}
}

func TestDatabaseRuleOpensToInternet(t *testing.T) {
	cases := []struct {
		name     string
		ruleType string
		value    string
		want     bool
	}{
		{"ip_addr any v4", "ip_addr", "0.0.0.0/0", true},
		{"ip_addr any v6", "ip_addr", "::/0", true},
		{"ip_addr specific", "ip_addr", "203.0.113.0/24", false},
		{"droplet any-looking value", "droplet", "0.0.0.0/0", false}, // type scopes to a droplet, not a CIDR
		{"tag", "tag", "web", false},
		{"k8s", "k8s", "cluster-uuid", false},
		{"app", "app", "app-uuid", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, databaseRuleOpensToInternet(c.ruleType, c.value))
		})
	}
}

func TestDatabaseTrustedSourcesAllowAny(t *testing.T) {
	cases := []struct {
		name  string
		rules []any
		want  bool
	}{
		{"no rules means open", nil, true},
		{"empty rules means open", []any{}, true},
		{
			"only specific ip rules",
			[]any{
				map[string]any{"type": "ip_addr", "value": "203.0.113.10/32"},
				map[string]any{"type": "droplet", "value": "12345"},
			},
			false,
		},
		{
			"contains any-address ip rule",
			[]any{
				map[string]any{"type": "ip_addr", "value": "203.0.113.10/32"},
				map[string]any{"type": "ip_addr", "value": "0.0.0.0/0"},
			},
			true,
		},
		{
			"contains any-address ipv6 rule",
			[]any{
				map[string]any{"type": "ip_addr", "value": "::/0"},
			},
			true,
		},
		{
			"only resource-scoped rules",
			[]any{
				map[string]any{"type": "tag", "value": "web"},
				map[string]any{"type": "k8s", "value": "cluster-uuid"},
			},
			false,
		},
		{
			"malformed entry ignored",
			[]any{
				"not-a-map",
				map[string]any{"type": "ip_addr", "value": "10.0.0.0/8"},
			},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, databaseTrustedSourcesAllowAny(c.rules))
		})
	}
}

func TestLoadBalancerFirewallAllowsAny(t *testing.T) {
	cases := []struct {
		name  string
		allow []any
		want  bool
	}{
		{"empty allow list permits all sources", nil, true},
		{"empty slice permits all sources", []any{}, true},
		{"specific allow CIDRs only", []any{"203.0.113.0/24", "198.51.100.7/32"}, false},
		{"allow includes any-address v4", []any{"203.0.113.0/24", "0.0.0.0/0"}, true},
		{"allow includes any-address v6", []any{"::/0"}, true},
		{"non-string entry ignored", []any{42, "10.0.0.0/8"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, loadBalancerFirewallAllowsAny(c.allow))
		})
	}
}

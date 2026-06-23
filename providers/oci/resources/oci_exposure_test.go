// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestOciCidrIsAny(t *testing.T) {
	cases := []struct {
		cidr string
		want bool
	}{
		{"0.0.0.0/0", true},
		{"::/0", true},
		{" 0.0.0.0/0 ", true},
		{"10.0.0.0/8", false},
		{"0.0.0.0", false}, // bare wildcard is not a CIDR route
		{"", false},
	}
	for _, c := range cases {
		if got := ociCidrIsAny(c.cidr); got != c.want {
			t.Errorf("ociCidrIsAny(%q) = %v, want %v", c.cidr, got, c.want)
		}
	}
}

func TestOciNsgRuleOpensIngress(t *testing.T) {
	cases := []struct {
		name string
		rule map[string]any
		want bool
	}{
		{"ingress cidr any", map[string]any{"direction": "INGRESS", "sourceType": "CIDR_BLOCK", "source": "0.0.0.0/0"}, true},
		{"ingress cidr any v6", map[string]any{"direction": "INGRESS", "sourceType": "CIDR_BLOCK", "source": "::/0"}, true},
		{"ingress cidr specific", map[string]any{"direction": "INGRESS", "sourceType": "CIDR_BLOCK", "source": "1.2.3.4/32"}, false},
		{"egress cidr any", map[string]any{"direction": "EGRESS", "sourceType": "CIDR_BLOCK", "source": "0.0.0.0/0"}, false},
		{"ingress nsg source", map[string]any{"direction": "INGRESS", "sourceType": "NETWORK_SECURITY_GROUP", "source": "ocid1.nsg"}, false},
		{"ingress service source", map[string]any{"direction": "INGRESS", "sourceType": "SERVICE_CIDR_BLOCK", "source": "all-services"}, false},
		{"missing sourceType but any cidr", map[string]any{"direction": "INGRESS", "source": "0.0.0.0/0"}, true},
		{"empty", map[string]any{}, false},
	}
	for _, c := range cases {
		if got := ociNsgRuleOpensIngress(c.rule); got != c.want {
			t.Errorf("ociNsgRuleOpensIngress(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOciWhitelistOpensInternet(t *testing.T) {
	cases := []struct {
		name   string
		ranges []any
		want   bool
	}{
		{"contains any cidr", []any{"1.2.3.4", "0.0.0.0/0"}, true},
		{"contains bare wildcard", []any{"0.0.0.0"}, true},
		{"contains v6 any", []any{"::/0"}, true},
		{"only specific", []any{"1.2.3.4", "10.0.0.0/8"}, false},
		{"empty denies (ACL on)", []any{}, false},
		{"non-string entries ignored", []any{42, "1.2.3.4"}, false},
	}
	for _, c := range cases {
		if got := ociWhitelistOpensInternet(c.ranges); got != c.want {
			t.Errorf("ociWhitelistOpensInternet(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

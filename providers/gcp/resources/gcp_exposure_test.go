// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestIsOpenCIDR(t *testing.T) {
	tests := []struct {
		cidr string
		want bool
	}{
		{"0.0.0.0/0", true},
		{"::/0", true},
		{" 0.0.0.0/0 ", true},
		{"10.0.0.0/8", false},
		{"192.168.1.0/24", false},
		{"0.0.0.0/32", false},
		{"", false},
		{"2001:db8::/32", false},
	}
	for _, tt := range tests {
		if got := isOpenCIDR(tt.cidr); got != tt.want {
			t.Errorf("isOpenCIDR(%q) = %v, want %v", tt.cidr, got, tt.want)
		}
	}
}

func TestGkeControlPlaneInternetReachable(t *testing.T) {
	tests := []struct {
		name           string
		publicEndpoint bool
		manEnforced    bool
		cidrs          []string
		want           bool
	}{
		{
			name:           "private endpoint is never reachable",
			publicEndpoint: false,
			manEnforced:    false,
			cidrs:          nil,
			want:           false,
		},
		{
			name:           "private endpoint with open authorized cidr still not reachable",
			publicEndpoint: false,
			manEnforced:    true,
			cidrs:          []string{"0.0.0.0/0"},
			want:           false,
		},
		{
			name:           "public endpoint without authorized networks is reachable",
			publicEndpoint: true,
			manEnforced:    false,
			cidrs:          nil,
			want:           true,
		},
		{
			name:           "public endpoint restricted to specific cidrs is not reachable",
			publicEndpoint: true,
			manEnforced:    true,
			cidrs:          []string{"203.0.113.0/24", "10.0.0.0/8"},
			want:           false,
		},
		{
			name:           "public endpoint with open ipv4 cidr in allowlist is reachable",
			publicEndpoint: true,
			manEnforced:    true,
			cidrs:          []string{"203.0.113.0/24", "0.0.0.0/0"},
			want:           true,
		},
		{
			name:           "public endpoint with open ipv6 cidr in allowlist is reachable",
			publicEndpoint: true,
			manEnforced:    true,
			cidrs:          []string{"::/0"},
			want:           true,
		},
		{
			name:           "public endpoint with empty enforced allowlist is not reachable",
			publicEndpoint: true,
			manEnforced:    true,
			cidrs:          nil,
			want:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gkeControlPlaneInternetReachable(tt.publicEndpoint, tt.manEnforced, tt.cidrs)
			if got != tt.want {
				t.Errorf("gkeControlPlaneInternetReachable(%v, %v, %v) = %v, want %v",
					tt.publicEndpoint, tt.manEnforced, tt.cidrs, got, tt.want)
			}
		})
	}
}

func TestFirewallRuleOpenIngress(t *testing.T) {
	cases := []struct {
		name      string
		direction string
		disabled  bool
		sources   []any
		want      bool
	}{
		{"ingress any v4", "INGRESS", false, []any{"0.0.0.0/0"}, true},
		{"ingress any v6", "INGRESS", false, []any{"::/0"}, true},
		{"ingress lowercase", "ingress", false, []any{"0.0.0.0/0"}, true},
		{"disabled", "INGRESS", true, []any{"0.0.0.0/0"}, false},
		{"egress", "EGRESS", false, []any{"0.0.0.0/0"}, false},
		{"scoped source", "INGRESS", false, []any{"10.0.0.0/8"}, false},
		{"no sources", "INGRESS", false, []any{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firewallRuleOpenIngress(c.direction, c.disabled, c.sources); got != c.want {
				t.Errorf("firewallRuleOpenIngress(%q,%v,%v) = %v, want %v", c.direction, c.disabled, c.sources, got, c.want)
			}
		})
	}
}

func TestNetworkNameFromUrl(t *testing.T) {
	cases := map[string]string{
		"https://www.googleapis.com/compute/v1/projects/p/global/networks/default": "default",
		"projects/p/global/networks/my-vpc":                                        "my-vpc",
		"default":                                                                  "default",
		"":                                                                         "",
	}
	for in, want := range cases {
		if got := networkNameFromUrl(in); got != want {
			t.Errorf("networkNameFromUrl(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirewallTargetsInstance(t *testing.T) {
	tags := map[string]bool{"web": true}
	sas := map[string]bool{"sa@project.iam.gserviceaccount.com": true}

	cases := []struct {
		name       string
		targetTags []any
		targetSAs  []any
		want       bool
	}{
		{"no targets applies to all", []any{}, []any{}, true},
		{"matching tag", []any{"web"}, []any{}, true},
		{"non-matching tag", []any{"db"}, []any{}, false},
		{"matching service account", []any{}, []any{"sa@project.iam.gserviceaccount.com"}, true},
		{"non-matching service account", []any{}, []any{"other@project.iam.gserviceaccount.com"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firewallTargetsInstance(c.targetTags, c.targetSAs, tags, sas); got != c.want {
				t.Errorf("firewallTargetsInstance(%v,%v) = %v, want %v", c.targetTags, c.targetSAs, got, c.want)
			}
		})
	}
}

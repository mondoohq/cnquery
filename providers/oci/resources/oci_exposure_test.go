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

func TestOciNsgIngressVerdict(t *testing.T) {
	open := map[string]any{"direction": "INGRESS", "sourceType": "CIDR_BLOCK", "source": "0.0.0.0/0"}
	specific := map[string]any{"direction": "INGRESS", "sourceType": "CIDR_BLOCK", "source": "1.2.3.4/32"}

	cases := []struct {
		name          string
		sets          [][]map[string]any
		wantAllows    bool
		wantOpenCount int
	}{
		{"no NSG attached is open", nil, true, 0},
		{"empty outer slice is open", [][]map[string]any{}, true, 0},
		{"one NSG with empty rule list is closed", [][]map[string]any{{}}, false, 0},
		{"one NSG only specific rules is closed", [][]map[string]any{{specific}}, false, 0},
		{"one NSG with an any-address rule is open", [][]map[string]any{{open}}, true, 1},
		{"two NSGs one empty one open is open", [][]map[string]any{{}, {open}}, true, 1},
		{"two NSGs both closed is closed", [][]map[string]any{{specific}, {}}, false, 0},
	}
	for _, c := range cases {
		openRules, allows := ociNsgIngressVerdict(c.sets)
		if allows != c.wantAllows {
			t.Errorf("ociNsgIngressVerdict(%s) allows = %v, want %v", c.name, allows, c.wantAllows)
		}
		if len(openRules) != c.wantOpenCount {
			t.Errorf("ociNsgIngressVerdict(%s) openRules len = %d, want %d", c.name, len(openRules), c.wantOpenCount)
		}
	}
}

func TestOciSecurityListRuleOpensIngress(t *testing.T) {
	cases := []struct {
		name string
		rule map[string]any
		want bool
	}{
		{"any cidr", map[string]any{"source_type": "CIDR_BLOCK", "source": "0.0.0.0/0"}, true},
		{"any cidr v6", map[string]any{"source_type": "CIDR_BLOCK", "source": "::/0"}, true},
		{"specific cidr", map[string]any{"source_type": "CIDR_BLOCK", "source": "1.2.3.4/32"}, false},
		{"service source", map[string]any{"source_type": "SERVICE_CIDR_BLOCK", "source": "all-services"}, false},
		{"missing source_type but any cidr", map[string]any{"source": "0.0.0.0/0"}, true},
		{"empty", map[string]any{}, false},
	}
	for _, c := range cases {
		if got := ociSecurityListRuleOpensIngress(c.rule); got != c.want {
			t.Errorf("ociSecurityListRuleOpensIngress(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOciCollectOpenSecurityListRulesEmptyIsOpen(t *testing.T) {
	// No security list resolvable falls back to OCI's default open posture,
	// mirroring the network security group "no firewall == open" convention.
	openRules, allows, err := ociCollectOpenSecurityListRules(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allows {
		t.Errorf("empty security list set should admit ingress (absent == open)")
	}
	if len(openRules) != 0 {
		t.Errorf("empty security list set should surface no open rules, got %d", len(openRules))
	}
}

func TestOciIngressOpen(t *testing.T) {
	cases := []struct {
		name           string
		nsgOpenRules   int
		securityListOK bool
		want           bool
	}{
		{"nsg opens", 2, false, true},
		{"security list opens", 0, true, true},
		{"both open", 1, true, true},
		{"neither opens", 0, false, false},
	}
	for _, c := range cases {
		if got := ociIngressOpen(c.nsgOpenRules, c.securityListOK); got != c.want {
			t.Errorf("ociIngressOpen(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOciAnySubnetReachable(t *testing.T) {
	cases := []struct {
		name  string
		gates []ociSubnetGate
		want  bool
	}{
		{"no subnets", nil, false},
		{"single subnet permits and routes", []ociSubnetGate{{prohibitsIngress: false, routesToInternet: true}}, true},
		{"single subnet permits but no route", []ociSubnetGate{{prohibitsIngress: false, routesToInternet: false}}, false},
		{"single subnet routes but prohibits", []ociSubnetGate{{prohibitsIngress: true, routesToInternet: true}}, false},
		// Regression: independent aggregation would have combined subnet A's
		// ingress with subnet B's route into a false positive. The conjunction is
		// per subnet, so neither subnet alone makes the resource reachable.
		{
			"permit-only subnet plus route-only subnet is not reachable",
			[]ociSubnetGate{
				{prohibitsIngress: false, routesToInternet: false},
				{prohibitsIngress: true, routesToInternet: true},
			},
			false,
		},
		{
			"one fully reachable subnet among others is reachable",
			[]ociSubnetGate{
				{prohibitsIngress: true, routesToInternet: true},
				{prohibitsIngress: false, routesToInternet: true},
			},
			true,
		},
	}
	for _, c := range cases {
		if got := ociAnySubnetReachable(c.gates); got != c.want {
			t.Errorf("ociAnySubnetReachable(%s) = %v, want %v", c.name, got, c.want)
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

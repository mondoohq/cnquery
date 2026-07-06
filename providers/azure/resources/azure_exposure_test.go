// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInternetOpenSourcePrefix(t *testing.T) {
	cases := []struct {
		prefix string
		want   bool
	}{
		{"*", true},
		{"0.0.0.0/0", true},
		{"::/0", true},
		{"Internet", true},
		{"internet", true},
		{" * ", true},
		{"10.0.0.0/8", false},
		{"VirtualNetwork", false},
		{"203.0.113.4", false},
		{"", false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, isInternetOpenSourcePrefix(c.prefix), "prefix %q", c.prefix)
	}
}

func TestSecurityRuleAllowsInternetIngress(t *testing.T) {
	t.Run("inbound allow from any is open", func(t *testing.T) {
		assert.True(t, securityRuleAllowsInternetIngress("Inbound", "Allow", "0.0.0.0/0", nil))
	})
	t.Run("inbound allow via service tag is open", func(t *testing.T) {
		assert.True(t, securityRuleAllowsInternetIngress("Inbound", "Allow", "Internet", nil))
	})
	t.Run("inbound allow via prefix list is open", func(t *testing.T) {
		assert.True(t, securityRuleAllowsInternetIngress("Inbound", "Allow", "10.0.0.0/8", []string{"VirtualNetwork", "*"}))
	})
	t.Run("outbound is not ingress", func(t *testing.T) {
		assert.False(t, securityRuleAllowsInternetIngress("Outbound", "Allow", "*", nil))
	})
	t.Run("deny is not open", func(t *testing.T) {
		assert.False(t, securityRuleAllowsInternetIngress("Inbound", "Deny", "*", nil))
	})
	t.Run("scoped source is not open", func(t *testing.T) {
		assert.False(t, securityRuleAllowsInternetIngress("Inbound", "Allow", "10.0.0.0/8", []string{"172.16.0.0/12"}))
	})
	t.Run("case insensitive direction/access", func(t *testing.T) {
		assert.True(t, securityRuleAllowsInternetIngress("inbound", "allow", "*", nil))
	})
}

func TestEffectiveRuleAllowsInternetIngress(t *testing.T) {
	t.Run("inbound allow from any prefix is open", func(t *testing.T) {
		assert.True(t, effectiveRuleAllowsInternetIngress(map[string]any{
			"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "0.0.0.0/0",
		}))
	})
	t.Run("inbound allow via Internet service tag is open", func(t *testing.T) {
		assert.True(t, effectiveRuleAllowsInternetIngress(map[string]any{
			"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "Internet",
		}))
	})
	t.Run("inbound allow via expanded prefixes is open", func(t *testing.T) {
		assert.True(t, effectiveRuleAllowsInternetIngress(map[string]any{
			"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "VirtualNetwork",
			"expandedSourceAddressPrefix": []any{"10.0.0.0/8", "*"},
		}))
	})
	t.Run("outbound is not ingress", func(t *testing.T) {
		assert.False(t, effectiveRuleAllowsInternetIngress(map[string]any{
			"direction": "Outbound", "access": "Allow", "sourceAddressPrefix": "*",
		}))
	})
	t.Run("deny is not open", func(t *testing.T) {
		assert.False(t, effectiveRuleAllowsInternetIngress(map[string]any{
			"direction": "Inbound", "access": "Deny", "sourceAddressPrefix": "*",
		}))
	})
	t.Run("scoped source is not open", func(t *testing.T) {
		assert.False(t, effectiveRuleAllowsInternetIngress(map[string]any{
			"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "10.0.0.0/8",
		}))
	})
	t.Run("missing keys are not open", func(t *testing.T) {
		assert.False(t, effectiveRuleAllowsInternetIngress(map[string]any{}))
	})
}

func TestPublicNetworkAccessEnabled(t *testing.T) {
	assert.True(t, publicNetworkAccessEnabled("Enabled"))
	assert.True(t, publicNetworkAccessEnabled(""), "empty defaults to enabled")
	assert.True(t, publicNetworkAccessEnabled("enabled"))
	assert.False(t, publicNetworkAccessEnabled("Disabled"))
	assert.False(t, publicNetworkAccessEnabled("disabled"))
}

func TestFirewallRuleAllowsAnyInternet(t *testing.T) {
	cases := []struct {
		start, end string
		want       bool
	}{
		{"0.0.0.0", "0.0.0.0", false},            // allow-all-Azure-services rule, not the public internet
		{"0.0.0.0", "255.255.255.255", true},     // full IPv4 span
		{"0.0.0.0", "128.255.255.255", true},     // wide partial span starting at 0.0.0.0
		{"203.0.113.1", "203.0.113.10", false},   // scoped range
		{"10.0.0.0", "10.255.255.255", false},    // private span
		{"", "", false},                          // empty
		{"0.0.0.0", "", false},                   // open start but no end
		{" 0.0.0.0 ", " 255.255.255.255 ", true}, // trimmed
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, firewallRuleAllowsAnyInternet(c.start, c.end), "%s-%s", c.start, c.end)
	}
}

func TestDatabaseInternetReachable(t *testing.T) {
	t.Run("public access disabled is never reachable", func(t *testing.T) {
		assert.False(t, databaseInternetReachable("Disabled", [][2]string{{"0.0.0.0", "255.255.255.255"}}))
	})
	t.Run("public access enabled with open rule is reachable", func(t *testing.T) {
		assert.True(t, databaseInternetReachable("Enabled", [][2]string{{"0.0.0.0", "255.255.255.255"}}))
	})
	t.Run("allow-all-Azure-services rule alone is not reachable", func(t *testing.T) {
		assert.False(t, databaseInternetReachable("Enabled", [][2]string{{"0.0.0.0", "0.0.0.0"}}))
	})
	t.Run("public access enabled but only scoped rules is not reachable", func(t *testing.T) {
		assert.False(t, databaseInternetReachable("Enabled", [][2]string{{"203.0.113.1", "203.0.113.10"}}))
	})
	t.Run("no firewall rules is not reachable", func(t *testing.T) {
		assert.False(t, databaseInternetReachable("Enabled", nil))
	})
}

func TestAksApiServerInternetReachable(t *testing.T) {
	t.Run("public cluster with no allowlist is reachable", func(t *testing.T) {
		assert.True(t, aksApiServerInternetReachable(false, "Enabled", nil))
	})
	t.Run("private cluster is not reachable", func(t *testing.T) {
		assert.False(t, aksApiServerInternetReachable(true, "Enabled", nil))
	})
	t.Run("disabled public access is not reachable", func(t *testing.T) {
		assert.False(t, aksApiServerInternetReachable(false, "Disabled", nil))
	})
	t.Run("authorized IP ranges close exposure", func(t *testing.T) {
		assert.False(t, aksApiServerInternetReachable(false, "Enabled", []string{"203.0.113.0/24"}))
	})
	t.Run("empty publicNetworkAccess defaults to reachable", func(t *testing.T) {
		assert.True(t, aksApiServerInternetReachable(false, "", nil))
	})
}

func TestStorageAccountIsPublic(t *testing.T) {
	t.Run("all gates open is public", func(t *testing.T) {
		assert.True(t, storageAccountIsPublic("Enabled", "Allow", true))
	})
	t.Run("blob public access off is not public", func(t *testing.T) {
		assert.False(t, storageAccountIsPublic("Enabled", "Allow", false))
	})
	t.Run("deny default action is not public", func(t *testing.T) {
		assert.False(t, storageAccountIsPublic("Enabled", "Deny", true))
	})
	t.Run("disabled public network access is not public", func(t *testing.T) {
		assert.False(t, storageAccountIsPublic("Disabled", "Allow", true))
	})
	t.Run("empty default action is treated as not Allow", func(t *testing.T) {
		assert.False(t, storageAccountIsPublic("Enabled", "", true))
	})
}

func TestRuleSourceIsInternet(t *testing.T) {
	assert.True(t, ruleSourceIsInternet(map[string]any{"sourceAddressPrefix": "*"}))
	assert.True(t, ruleSourceIsInternet(map[string]any{"sourceAddressPrefix": "Internet"}))
	assert.True(t, ruleSourceIsInternet(map[string]any{"sourceAddressPrefixes": []any{"10.0.0.0/8", "0.0.0.0/0"}}))
	assert.True(t, ruleSourceIsInternet(map[string]any{"expandedSourceAddressPrefix": []any{"*"}}))
	assert.False(t, ruleSourceIsInternet(map[string]any{"sourceAddressPrefix": "VirtualNetwork"}))
	assert.False(t, ruleSourceIsInternet(map[string]any{"sourceAddressPrefixes": []any{"10.0.0.0/8"}}))
	assert.False(t, ruleSourceIsInternet(map[string]any{}))
}

func TestRuleInt(t *testing.T) {
	v, ok := ruleInt(map[string]any{"priority": float64(100)}, "priority")
	assert.True(t, ok)
	assert.Equal(t, 100, v)
	v, ok = ruleInt(map[string]any{"priority": 200}, "priority")
	assert.True(t, ok)
	assert.Equal(t, 200, v)
	_, ok = ruleInt(map[string]any{"priority": "300"}, "priority")
	assert.False(t, ok, "string priority is not accepted")
	_, ok = ruleInt(map[string]any{}, "priority")
	assert.False(t, ok)
}

func TestRulePortIntervals(t *testing.T) {
	assert.Equal(t, []portInterval{{0, 65535}}, rulePortIntervals(map[string]any{"destinationPortRange": "*"}))
	assert.Equal(t, []portInterval{{22, 22}}, rulePortIntervals(map[string]any{"destinationPortRange": "22"}))
	assert.Equal(t, []portInterval{{80, 443}}, rulePortIntervals(map[string]any{"destinationPortRange": "80-443"}))
	assert.Equal(t, []portInterval{{22, 22}, {443, 443}}, rulePortIntervals(map[string]any{"destinationPortRanges": []any{"22", "443"}}))
	assert.Equal(t, []portInterval{{0, 65535}}, rulePortIntervals(map[string]any{}), "absent ports means all ports")
}

func TestPortsCover(t *testing.T) {
	all := []portInterval{{0, 65535}}
	assert.True(t, portsCover(all, []portInterval{{22, 22}}))
	assert.True(t, portsCover([]portInterval{{20, 30}}, []portInterval{{22, 25}}))
	assert.False(t, portsCover([]portInterval{{20, 30}}, []portInterval{{22, 40}}))
	assert.False(t, portsCover([]portInterval{{80, 80}}, []portInterval{{22, 22}}))
	// allow spanning two deny intervals is not covered (must fall within one)
	assert.False(t, portsCover([]portInterval{{20, 25}, {26, 30}}, []portInterval{{22, 28}}))
}

func TestProtocolCovers(t *testing.T) {
	assert.True(t, protocolCovers("*", "Tcp"))
	assert.True(t, protocolCovers("Any", "Tcp"))
	assert.True(t, protocolCovers("", "Tcp"))
	assert.True(t, protocolCovers("Tcp", "tcp"))
	assert.False(t, protocolCovers("Udp", "Tcp"))
}

func TestDestCovers(t *testing.T) {
	assert.True(t, destCovers(map[string]any{"destinationAddressPrefix": "*"}, map[string]any{"destinationAddressPrefix": "10.0.0.4"}))
	assert.True(t, destCovers(map[string]any{"destinationAddressPrefix": "0.0.0.0/0"}, map[string]any{"destinationAddressPrefix": "10.0.0.4"}))
	assert.True(t, destCovers(map[string]any{"destinationAddressPrefix": "10.0.0.4"}, map[string]any{"destinationAddressPrefix": "10.0.0.4"}))
	assert.True(t, destCovers(map[string]any{"destinationAddressPrefixes": []any{"10.0.0.4"}}, map[string]any{"destinationAddressPrefix": "10.0.0.4"}))
	assert.False(t, destCovers(map[string]any{"destinationAddressPrefix": "10.0.0.5"}, map[string]any{"destinationAddressPrefix": "10.0.0.4"}))
	// allow rule using the plural destination form is read on the allow side too
	assert.True(t, destCovers(map[string]any{"destinationAddressPrefix": "10.0.0.4"}, map[string]any{"destinationAddressPrefixes": []any{"10.0.0.4"}}))
	// every allow destination must be covered; a deny covering only one is not enough
	assert.False(t, destCovers(map[string]any{"destinationAddressPrefix": "10.0.0.4"}, map[string]any{"destinationAddressPrefixes": []any{"10.0.0.4", "10.0.0.5"}}))
	assert.True(t, destCovers(map[string]any{"destinationAddressPrefix": "*"}, map[string]any{"destinationAddressPrefixes": []any{"10.0.0.4", "10.0.0.5"}}))
	// a narrow deny cannot cover an allow that targets all addresses
	assert.False(t, destCovers(map[string]any{"destinationAddressPrefix": "10.0.0.4"}, map[string]any{"destinationAddressPrefix": "*"}))
}

func TestDenyDominatesAllow(t *testing.T) {
	denyAll := map[string]any{"protocol": "*", "destinationPortRange": "*", "destinationAddressPrefix": "*"}
	allowSsh := map[string]any{"protocol": "Tcp", "destinationPortRange": "22", "destinationAddressPrefix": "*"}
	assert.True(t, denyDominatesAllow(denyAll, allowSsh))

	denyOtherPort := map[string]any{"protocol": "*", "destinationPortRange": "80", "destinationAddressPrefix": "*"}
	assert.False(t, denyDominatesAllow(denyOtherPort, allowSsh), "deny on a different port does not shadow")

	denyOtherProto := map[string]any{"protocol": "Udp", "destinationPortRange": "*", "destinationAddressPrefix": "*"}
	assert.False(t, denyDominatesAllow(denyOtherProto, allowSsh), "deny on a different protocol does not shadow")

	denyOtherDest := map[string]any{"protocol": "*", "destinationPortRange": "*", "destinationAddressPrefix": "10.0.0.5"}
	assert.False(t, denyDominatesAllow(denyOtherDest, allowSsh), "deny on a different destination does not shadow")
}

func TestNsgAllowsInternetIngress(t *testing.T) {
	allowSsh := map[string]any{
		"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "*",
		"protocol": "Tcp", "destinationPortRange": "22", "priority": float64(300),
		"destinationAddressPrefix": "*",
	}
	denyAllHigh := map[string]any{
		"direction": "Inbound", "access": "Deny", "sourceAddressPrefix": "*",
		"protocol": "*", "destinationPortRange": "*", "priority": float64(100),
		"destinationAddressPrefix": "*",
	}
	denyAllLow := map[string]any{
		"direction": "Inbound", "access": "Deny", "sourceAddressPrefix": "*",
		"protocol": "*", "destinationPortRange": "*", "priority": float64(4000),
		"destinationAddressPrefix": "*",
	}

	t.Run("allow with no deny is open", func(t *testing.T) {
		open, surviving := nsgAllowsInternetIngress([]map[string]any{allowSsh})
		assert.True(t, open)
		assert.Len(t, surviving, 1)
	})
	t.Run("higher-priority deny-all shadows the allow", func(t *testing.T) {
		open, surviving := nsgAllowsInternetIngress([]map[string]any{allowSsh, denyAllHigh})
		assert.False(t, open)
		assert.Empty(t, surviving)
	})
	t.Run("lower-priority deny-all does not shadow the allow", func(t *testing.T) {
		open, surviving := nsgAllowsInternetIngress([]map[string]any{allowSsh, denyAllLow})
		assert.True(t, open)
		assert.Len(t, surviving, 1)
	})
	t.Run("deny on a different port leaves the allow open", func(t *testing.T) {
		denyHttp := map[string]any{
			"direction": "Inbound", "access": "Deny", "sourceAddressPrefix": "*",
			"protocol": "Tcp", "destinationPortRange": "80", "priority": float64(100),
			"destinationAddressPrefix": "*",
		}
		open, _ := nsgAllowsInternetIngress([]map[string]any{allowSsh, denyHttp})
		assert.True(t, open)
	})
	t.Run("no internet-source rule admits nothing", func(t *testing.T) {
		vnetAllow := map[string]any{
			"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "VirtualNetwork",
			"protocol": "*", "destinationPortRange": "*", "priority": float64(100),
		}
		open, surviving := nsgAllowsInternetIngress([]map[string]any{vnetAllow})
		assert.False(t, open)
		assert.Empty(t, surviving)
	})
	t.Run("deny-internet only admits nothing", func(t *testing.T) {
		open, _ := nsgAllowsInternetIngress([]map[string]any{denyAllHigh})
		assert.False(t, open)
	})
	t.Run("one allow shadowed, another open", func(t *testing.T) {
		allowHttp := map[string]any{
			"direction": "Inbound", "access": "Allow", "sourceAddressPrefix": "*",
			"protocol": "Tcp", "destinationPortRange": "443", "priority": float64(200),
			"destinationAddressPrefix": "*",
		}
		denySshHigh := map[string]any{
			"direction": "Inbound", "access": "Deny", "sourceAddressPrefix": "*",
			"protocol": "Tcp", "destinationPortRange": "22", "priority": float64(150),
			"destinationAddressPrefix": "*",
		}
		open, surviving := nsgAllowsInternetIngress([]map[string]any{allowSsh, allowHttp, denySshHigh})
		assert.True(t, open)
		assert.Len(t, surviving, 1, "only the HTTPS allow survives")
	})
}

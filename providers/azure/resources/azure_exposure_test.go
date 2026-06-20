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

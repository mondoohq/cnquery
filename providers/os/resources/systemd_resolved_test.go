// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResolvedConfCache(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		fallback bool
		want     bool
	}{
		{"cache disabled", "[Resolve]\nCache=no\n", true, false},
		{"cache enabled", "[Resolve]\nCache=yes\n", false, true},
		{"no-negative still caches", "[Resolve]\nCache=no-negative\n", false, true},
		{"case-insensitive key and value", "[Resolve]\n  cache = NO \n", true, false},
		{"commented out keeps fallback", "[Resolve]\n#Cache=no\n", true, true},
		{"absent keeps fallback", "[Resolve]\nDNSSEC=yes\n", true, true},
		{"last assignment wins", "[Resolve]\nCache=yes\nCache=no\n", true, false},
		{"cache outside resolve section ignored", "[DHCPv4]\nCache=no\n", true, true},
		{"cache after leaving resolve section ignored", "[Resolve]\nDNSSEC=yes\n[Network]\nCache=no\n", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseResolvedConfCache(strings.NewReader(tc.content), tc.fallback)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseResolvectlGlobal_Basic(t *testing.T) {
	input := `Global
         Protocols: -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
  resolv.conf mode: stub
Current DNS Server: 1.1.1.1
       DNS Servers: 1.1.1.1 1.0.0.1
        DNS Domain: corp.example.com ~example.com

Link 2 (eth0)
    Current Scopes: DNS
         Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS
Current DNS Server: 192.168.1.1
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)

	assert.Equal(t, []string{"1.1.1.1", "1.0.0.1"}, g.dns)
	assert.Equal(t, "1.1.1.1", g.currentDnsServer)
	assert.Equal(t, []string{"corp.example.com", "~example.com"}, g.domains)
	assert.Equal(t, "stub", g.resolvConfMode)
	assert.Equal(t, "no/unsupported", g.dnssec)
	assert.Equal(t, "no", g.llmnr)
	assert.Equal(t, "no", g.multicastDns)
	assert.Equal(t, "no", g.dnsOverTls)
	assert.True(t, g.cache, "cache defaults to true when no explicit field is present")

	// Per-link DNS server should NOT leak into the global block.
	for _, addr := range g.dns {
		assert.NotEqual(t, "192.168.1.1", addr)
	}
}

func TestParseResolvectlGlobal_CacheDisabled(t *testing.T) {
	// systemd versions that emit a `Cache:` line must override the default.
	input := `Global
         Protocols: -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
             Cache: no
       DNS Servers: 1.1.1.1
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)

	assert.False(t, g.cache, "explicit `Cache: no` overrides default of true")
}

func TestParseResolvectlGlobal_CacheEnabledExplicit(t *testing.T) {
	input := `Global
             Cache: yes
       DNS Servers: 1.1.1.1
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)

	assert.True(t, g.cache, "explicit `Cache: yes` matches the default")
}

func TestParseResolvectlGlobal_CurrentDnsServerOnly(t *testing.T) {
	// Some hosts only have a Current DNS Server line (e.g., when no static
	// global DNS is configured but resolvectl reports the active selection).
	input := `Global
         Protocols: -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 9.9.9.9
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)

	assert.Equal(t, "9.9.9.9", g.currentDnsServer)
	assert.Empty(t, g.dns, "no DNS Servers line means dns stays empty")
}

func TestParseResolvectlGlobal_PositiveProtocols(t *testing.T) {
	input := `Global
         Protocols: +LLMNR +mDNS +DNSOverTLS DNSSEC=yes
  resolv.conf mode: static
       DNS Servers: 9.9.9.9
Fallback DNS Servers: 8.8.8.8 8.8.4.4
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)

	assert.Equal(t, "yes", g.llmnr)
	assert.Equal(t, "yes", g.multicastDns)
	assert.Equal(t, "yes", g.dnsOverTls)
	assert.Equal(t, "yes", g.dnssec)
	assert.Equal(t, "static", g.resolvConfMode)
	assert.Equal(t, []string{"9.9.9.9"}, g.dns)
	assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, g.fallbackDns)
}

func TestParseResolvectlGlobal_KeyValueProtocols(t *testing.T) {
	// Some resolvectl versions render protocols as full KEY=VALUE tokens.
	input := `Global
         Protocols: LLMNR=resolve MulticastDNS=resolve DNSOverTLS=opportunistic DNSSEC=allow-downgrade
       DNS Servers: 1.1.1.1
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)

	assert.Equal(t, "resolve", g.llmnr)
	assert.Equal(t, "resolve", g.multicastDns)
	assert.Equal(t, "opportunistic", g.dnsOverTls)
	assert.Equal(t, "allow-downgrade", g.dnssec)
}

func TestParseResolvectlGlobal_EmptyInput(t *testing.T) {
	g := &resolvedGlobal{}
	parseResolvectlGlobal("", g)
	assert.Empty(t, g.dns)
	assert.Empty(t, g.fallbackDns)
	assert.True(t, g.cache, "default cache is true")
}

func TestParseResolvectlGlobal_StopsAtBlankLine(t *testing.T) {
	// Once the Global block ends, subsequent lines must not be parsed even
	// if they look like Global fields (e.g. per-link "DNS Servers").
	input := `Global
       DNS Servers: 1.2.3.4

       DNS Servers: 9.9.9.9
`
	g := &resolvedGlobal{}
	parseResolvectlGlobal(input, g)
	assert.Equal(t, []string{"1.2.3.4"}, g.dns)
}

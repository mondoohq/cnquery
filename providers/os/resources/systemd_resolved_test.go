// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

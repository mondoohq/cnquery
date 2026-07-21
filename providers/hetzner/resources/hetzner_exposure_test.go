// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
)

func TestFirewallRuleOpenToInternet(t *testing.T) {
	tests := []struct {
		name string
		rule map[string]any
		want bool
	}{
		{"inbound any ipv4", map[string]any{"direction": "in", "sourceIps": []any{"0.0.0.0/0"}}, true},
		{"inbound any ipv6", map[string]any{"direction": "in", "sourceIps": []any{"::/0"}}, true},
		{"inbound mixed with any", map[string]any{"direction": "in", "sourceIps": []any{"10.0.0.0/8", "0.0.0.0/0"}}, true},
		{"inbound scoped source", map[string]any{"direction": "in", "sourceIps": []any{"203.0.113.0/24"}}, false},
		{"outbound any", map[string]any{"direction": "out", "sourceIps": []any{"0.0.0.0/0"}}, false},
		{"no sources", map[string]any{"direction": "in"}, false},
		{"empty", map[string]any{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, firewallRuleOpenToInternet(tt.rule))
		})
	}
}

func TestLoadBalancerHasPublicIp(t *testing.T) {
	ipv4 := net.ParseIP("203.0.113.10")
	ipv6 := net.ParseIP("2001:db8::1")
	tests := []struct {
		name string
		pn   hcloud.LoadBalancerPublicNet
		want bool
	}{
		{"disabled with ipv4", hcloud.LoadBalancerPublicNet{Enabled: false, IPv4: hcloud.LoadBalancerPublicNetIPv4{IP: ipv4}}, false},
		{"enabled with ipv4", hcloud.LoadBalancerPublicNet{Enabled: true, IPv4: hcloud.LoadBalancerPublicNetIPv4{IP: ipv4}}, true},
		{"enabled with ipv6 only", hcloud.LoadBalancerPublicNet{Enabled: true, IPv6: hcloud.LoadBalancerPublicNetIPv6{IP: ipv6}}, true},
		{"enabled no ip", hcloud.LoadBalancerPublicNet{Enabled: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, loadBalancerHasPublicIp(tt.pn))
		})
	}
}

func TestLoadBalancerServiceDicts(t *testing.T) {
	got := loadBalancerServiceDicts([]hcloud.LoadBalancerService{
		{Protocol: hcloud.LoadBalancerServiceProtocolHTTPS, ListenPort: 443, DestinationPort: 8443},
	})
	assert.Len(t, got, 1)
	d := got[0].(map[string]any)
	assert.Equal(t, "https", d["protocol"])
	assert.Equal(t, int64(443), d["listenPort"])
	assert.Equal(t, int64(8443), d["destinationPort"])

	assert.Empty(t, loadBalancerServiceDicts(nil))
}

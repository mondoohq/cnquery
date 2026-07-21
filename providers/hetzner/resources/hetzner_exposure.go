// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// firewallRuleOpenToInternet reports whether a Hetzner firewall rule dict is an
// inbound rule whose source admits any address (0.0.0.0/0 or ::/0).
func firewallRuleOpenToInternet(rule map[string]any) bool {
	if direction, _ := rule["direction"].(string); direction != "in" {
		return false
	}
	sources, ok := rule["sourceIps"].([]any)
	if !ok {
		return false
	}
	for _, s := range sources {
		if cidr, ok := s.(string); ok && (cidr == "0.0.0.0/0" || cidr == "::/0") {
			return true
		}
	}
	return false
}

// exposure breaks down whether the server is reachable from the internet: a
// public IP combined with firewall ingress that admits any address. A server
// with no firewall attached is open, so an empty firewall set counts as
// admitting ingress.
func (s *mqlHetznerServer) exposure() (*mqlHetznerNetworkExposure, error) {
	id := s.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	ipv4 := s.GetPublicIpv4()
	if ipv4.Error != nil {
		return nil, ipv4.Error
	}
	ipv6 := s.GetPublicIpv6()
	if ipv6.Error != nil {
		return nil, ipv6.Error
	}
	hasPublicIp := ipv4.Data != "" || ipv6.Data != ""

	firewalls := s.GetFirewalls()
	if firewalls.Error != nil {
		return nil, firewalls.Error
	}
	openRules := []any{}
	for _, f := range firewalls.Data {
		fw, ok := f.(*mqlHetznerFirewall)
		if !ok {
			continue
		}
		rules := fw.GetRules()
		if rules.Error != nil {
			return nil, rules.Error
		}
		for _, r := range rules.Data {
			rule, ok := r.(map[string]any)
			if ok && firewallRuleOpenToInternet(rule) {
				openRules = append(openRules, rule)
			}
		}
	}

	firewallAllowsIngress := len(firewalls.Data) == 0 || len(openRules) > 0
	internetReachable := hasPublicIp && firewallAllowsIngress

	res, err := CreateResource(s.MqlRuntime, "hetzner.network.exposure", map[string]*llx.RawData{
		"__id":                  llx.StringData(fmt.Sprintf("hetzner.server/%d/exposure", id.Data)),
		"internetReachable":     llx.BoolData(internetReachable),
		"hasPublicIp":           llx.BoolData(hasPublicIp),
		"firewallAllowsIngress": llx.BoolData(firewallAllowsIngress),
		"openIngressRules":      llx.ArrayData(openRules, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerNetworkExposure), nil
}

// loadBalancerHasPublicIp reports whether a Hetzner load balancer's public
// network is enabled and carries a public IPv4 or IPv6 address.
func loadBalancerHasPublicIp(pn hcloud.LoadBalancerPublicNet) bool {
	if !pn.Enabled {
		return false
	}
	return pn.IPv4.IP != nil || pn.IPv6.IP != nil
}

// loadBalancerServiceDicts renders a load balancer's forwarding services as
// dicts describing the public listeners that admit traffic from any address.
func loadBalancerServiceDicts(services []hcloud.LoadBalancerService) []any {
	out := make([]any, 0, len(services))
	for _, s := range services {
		out = append(out, map[string]any{
			"protocol":        string(s.Protocol),
			"listenPort":      int64(s.ListenPort),
			"destinationPort": int64(s.DestinationPort),
		})
	}
	return out
}

// exposure breaks down whether the load balancer is reachable from the internet.
// A Hetzner load balancer has no firewall; its public network being enabled with
// a public IP plus at least one forwarding service makes it internet-reachable.
func (m *mqlHetznerLoadBalancer) exposure() (*mqlHetznerNetworkExposure, error) {
	id := m.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	hasPublicIp := loadBalancerHasPublicIp(m.cachePublicNet)
	openRules := loadBalancerServiceDicts(m.cacheServices)
	servicesAllowIngress := len(openRules) > 0
	internetReachable := hasPublicIp && servicesAllowIngress

	res, err := CreateResource(m.MqlRuntime, "hetzner.network.exposure", map[string]*llx.RawData{
		"__id":                  llx.StringData(fmt.Sprintf("hetzner.loadBalancer/%d/exposure", id.Data)),
		"internetReachable":     llx.BoolData(internetReachable),
		"hasPublicIp":           llx.BoolData(hasPublicIp),
		"firewallAllowsIngress": llx.BoolData(servicesAllowIngress),
		"openIngressRules":      llx.ArrayData(openRules, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerNetworkExposure), nil
}

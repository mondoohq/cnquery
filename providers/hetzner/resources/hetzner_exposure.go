// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

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

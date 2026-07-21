// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// anyAddressCIDRs are the CIDR notations that mean "every address" — an
// open-to-the-internet source for either IP stack.
var anyAddressCIDRs = map[string]struct{}{
	"0.0.0.0/0": {},
	"::/0":      {},
}

// isAnyAddress reports whether a CIDR/IP string admits every address. Bare
// "0.0.0.0" / "::" (without a prefix) are also treated as any-address since
// DigitalOcean accepts them as wildcard sources.
func isAnyAddress(cidr string) bool {
	c := strings.TrimSpace(cidr)
	if _, ok := anyAddressCIDRs[c]; ok {
		return true
	}
	return c == "0.0.0.0" || c == "::"
}

// databaseRuleOpensToInternet reports whether a single managed-database
// trusted-source rule admits traffic from any address. Only ip_addr rules can
// reference a CIDR; droplet/k8s/tag/app rules scope to specific resources and
// never open the cluster to the whole internet.
func databaseRuleOpensToInternet(ruleType, value string) bool {
	if ruleType != "ip_addr" {
		return false
	}
	return isAnyAddress(value)
}

// databaseTrustedSourcesAllowAny inspects a managed database's trusted-source
// (firewall) rules and reports whether the public endpoint is open to any
// address. When no rules are configured at all DigitalOcean leaves the public
// connection endpoint reachable from every address, so an empty rule set also
// counts as open.
func databaseTrustedSourcesAllowAny(rules []any) bool {
	if len(rules) == 0 {
		return true
	}
	for _, r := range rules {
		rule, ok := r.(map[string]any)
		if !ok {
			continue
		}
		ruleType, _ := rule["type"].(string)
		value, _ := rule["value"].(string)
		if databaseRuleOpensToInternet(ruleType, value) {
			return true
		}
	}
	return false
}

// loadBalancerFirewallAllowsAny reports whether a load balancer's source
// firewall admits traffic from any address. The load balancer firewall is an
// allow/deny list of source CIDRs: when the allow list is empty every source
// is permitted (subject to the deny list); when it is non-empty only those
// sources are permitted, so it is open to the internet only if one of the
// allow entries is itself an any-address CIDR.
func loadBalancerFirewallAllowsAny(allow []any) bool {
	if len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		cidr, ok := a.(string)
		if !ok {
			continue
		}
		if isAnyAddress(cidr) {
			return true
		}
	}
	return false
}

// exposure breaks down whether the droplet is reachable from the internet: a
// public IP combined with firewall ingress that admits any address. A droplet
// with no firewall attached is fully open, so missingFirewall counts as
// admitting ingress.
func (d *mqlDigitaloceanDroplet) exposure() (*mqlDigitaloceanNetworkExposure, error) {
	id := d.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	ipv4 := d.GetPublicIpv4()
	if ipv4.Error != nil {
		return nil, ipv4.Error
	}
	ipv6 := d.GetPublicIpv6()
	if ipv6.Error != nil {
		return nil, ipv6.Error
	}
	hasPublicIp := ipv4.Data != "" || ipv6.Data != ""

	missingFirewall := d.GetMissingFirewall()
	if missingFirewall.Error != nil {
		return nil, missingFirewall.Error
	}

	openRules := []any{}
	firewalls := d.GetFirewalls()
	if firewalls.Error != nil {
		return nil, firewalls.Error
	}
	for _, f := range firewalls.Data {
		fw, ok := f.(*mqlDigitaloceanFirewall)
		if !ok {
			continue
		}
		rules := fw.GetIngressRules()
		if rules.Error != nil {
			return nil, rules.Error
		}
		for _, r := range rules.Data {
			rule, ok := r.(*mqlDigitaloceanFirewallIngressRule)
			if !ok {
				continue
			}
			open := rule.GetOpenToInternet()
			if open.Error != nil {
				return nil, open.Error
			}
			if open.Data {
				openRules = append(openRules, rule)
			}
		}
	}

	firewallAllowsIngress := missingFirewall.Data || len(openRules) > 0
	internetReachable := hasPublicIp && firewallAllowsIngress

	res, err := CreateResource(d.MqlRuntime, "digitalocean.network.exposure", map[string]*llx.RawData{
		"__id":                  llx.StringData(fmt.Sprintf("digitalocean.droplet/%d/exposure", id.Data)),
		"internetReachable":     llx.BoolData(internetReachable),
		"hasPublicIp":           llx.BoolData(hasPublicIp),
		"firewallAllowsIngress": llx.BoolData(firewallAllowsIngress),
		"openIngressRules":      llx.ArrayData(openRules, types.Resource("digitalocean.firewall.ingressRule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDigitaloceanNetworkExposure), nil
}

// internetReachable reports whether a managed database cluster is reachable
// from the internet: it has a public connection endpoint and its trusted-source
// firewall rules admit any address (or no rules are configured, which leaves
// the public endpoint open to every address). Managed databases use an
// authorized-networks model rather than droplet-style cloud firewalls, so this
// is a single predicate rather than the full network.exposure breakdown.
func (d *mqlDigitaloceanDatabase) internetReachable() (bool, error) {
	host := d.GetConnectionHost()
	if host.Error != nil {
		return false, host.Error
	}
	// No public connection endpoint means the cluster is private-only.
	if host.Data == "" {
		return false, nil
	}

	rules := d.GetFirewallRules()
	if rules.Error != nil {
		return false, rules.Error
	}

	return databaseTrustedSourcesAllowAny(rules.Data), nil
}

// exposure breaks down whether a load balancer is reachable from the internet:
// an EXTERNAL (internet-facing) load balancer has a public IP, and its source
// firewall admits traffic from any address (an empty allow list, or an allow
// entry that is itself an any-address CIDR). INTERNAL load balancers serve
// VPC-only traffic and are not internet-reachable.
func (l *mqlDigitaloceanLoadBalancer) exposure() (*mqlDigitaloceanNetworkExposure, error) {
	id := l.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	network := l.GetNetwork()
	if network.Error != nil {
		return nil, network.Error
	}

	ipv4 := l.GetIp()
	if ipv4.Error != nil {
		return nil, ipv4.Error
	}
	ipv6 := l.GetIpv6()
	if ipv6.Error != nil {
		return nil, ipv6.Error
	}
	// An INTERNAL load balancer is VPC-only; treat its address as non-public
	// even though the API may still report an IP.
	internalOnly := strings.EqualFold(network.Data, "INTERNAL")
	hasPublicIp := !internalOnly && (ipv4.Data != "" || ipv6.Data != "")

	allow := l.GetFirewallAllow()
	if allow.Error != nil {
		return nil, allow.Error
	}
	firewallAllowsIngress := loadBalancerFirewallAllowsAny(allow.Data)

	internetReachable := hasPublicIp && firewallAllowsIngress

	res, err := CreateResource(l.MqlRuntime, "digitalocean.network.exposure", map[string]*llx.RawData{
		"__id":                  llx.StringData(fmt.Sprintf("digitalocean.loadBalancer/%s/exposure", id.Data)),
		"internetReachable":     llx.BoolData(internetReachable),
		"hasPublicIp":           llx.BoolData(hasPublicIp),
		"firewallAllowsIngress": llx.BoolData(firewallAllowsIngress),
		// openIngressRules are droplet cloud-firewall rules; load balancers use
		// a source allow/deny list instead, so this is always empty for them.
		"openIngressRules": llx.ArrayData([]any{}, types.Resource("digitalocean.firewall.ingressRule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDigitaloceanNetworkExposure), nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// ociCidrIsAny reports whether a CIDR string admits any address — the IPv4
// default route 0.0.0.0/0 or the IPv6 default route ::/0. Surrounding
// whitespace is tolerated.
func ociCidrIsAny(cidr string) bool {
	c := strings.TrimSpace(cidr)
	return c == "0.0.0.0/0" || c == "::/0"
}

// ociNsgRuleOpensIngress reports whether an OCI network-security-group rule dict
// is an INGRESS rule whose source is a CIDR block admitting any address. NSG and
// SERVICE_CIDR_BLOCK sources reference internal networks, not an internet-wide
// opening, so only CIDR_BLOCK sources can be public.
func ociNsgRuleOpensIngress(rule map[string]any) bool {
	if direction, _ := rule["direction"].(string); !strings.EqualFold(direction, "INGRESS") {
		return false
	}
	if st, _ := rule["sourceType"].(string); st != "" && !strings.EqualFold(st, "CIDR_BLOCK") {
		return false
	}
	source, _ := rule["source"].(string)
	return ociCidrIsAny(source)
}

// ociCollectOpenNsgRules inspects the INGRESS rules of the given network
// security group resources and returns the ones that admit traffic from any
// address, along with whether the NSG set admits ingress at all. With no NSG
// attached the resource falls back to its subnet's default security posture,
// which OCI leaves open, so an empty NSG set counts as admitting ingress (the
// "no firewall == open" convention shared with the other providers).
func ociCollectOpenNsgRules(nsgs []any) ([]any, bool, error) {
	ruleSets := make([][]map[string]any, 0, len(nsgs))
	for _, g := range nsgs {
		nsg, ok := g.(*mqlOciNetworkNetworkSecurityGroup)
		if !ok {
			continue
		}
		rules := nsg.GetIngressSecurityRules()
		if rules.Error != nil {
			return nil, false, rules.Error
		}
		set := make([]map[string]any, 0, len(rules.Data))
		for _, r := range rules.Data {
			if rule, ok := r.(map[string]any); ok {
				set = append(set, rule)
			}
		}
		ruleSets = append(ruleSets, set)
	}
	openRules, allowsIngress := ociNsgIngressVerdict(ruleSets)
	return openRules, allowsIngress, nil
}

// ociNsgIngressVerdict evaluates the ingress rules of a set of attached network
// security groups (one inner slice of rule dicts per NSG) and returns the rules
// that admit traffic from any address, plus whether the set admits ingress at
// all. No NSG attached (an empty outer slice) falls back to the subnet's default
// posture, which OCI leaves open, so it counts as admitting ingress. NSGs that
// are attached but whose rules never match — including NSGs with empty rule
// lists — are a deliberate lock-down and count as closed.
func ociNsgIngressVerdict(nsgRuleSets [][]map[string]any) ([]any, bool) {
	openRules := []any{}
	for _, rules := range nsgRuleSets {
		for _, rule := range rules {
			if ociNsgRuleOpensIngress(rule) {
				openRules = append(openRules, rule)
			}
		}
	}
	return openRules, len(nsgRuleSets) == 0 || len(openRules) > 0
}

// ociWhitelistOpensInternet reports whether an Autonomous Database access-control
// allow-list admits any address. Unlike a security-group rule set, an *empty*
// ADB allow-list with access control enabled denies everyone, so only an entry
// that is an any-address route (0.0.0.0/0, ::/0) or the bare wildcard 0.0.0.0
// counts as internet-open.
func ociWhitelistOpensInternet(ranges []any) bool {
	for _, r := range ranges {
		s, ok := r.(string)
		if !ok {
			continue
		}
		c := strings.TrimSpace(s)
		if c == "0.0.0.0" || ociCidrIsAny(c) {
			return true
		}
	}
	return false
}

// exposure breaks down whether the compute instance is reachable from the
// internet: a VNIC with a public IP, on a subnet that does not prohibit internet
// ingress, whose attached network security groups admit inbound from any address
// (or that has no NSG attached — OCI's default security list opens SSH to the
// internet, so an NSG-less VNIC on an unrestricted subnet is treated as open,
// matching the "no firewall == open" convention used by the other providers).
func (i *mqlOciComputeInstance) exposure() (*mqlOciNetworkExposure, error) {
	id := i.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	vnics := i.GetVnics()
	if vnics.Error != nil {
		return nil, vnics.Error
	}

	hasPublicIp := false
	securityGroupAllowsIngress := false
	internetReachable := false
	openRules := []any{}

	for _, v := range vnics.Data {
		vnic, ok := v.(*mqlOciComputeVnic)
		if !ok {
			continue
		}

		pub := vnic.GetPublicIp()
		if pub.Error != nil {
			return nil, pub.Error
		}
		vnicHasPublicIp := strings.TrimSpace(pub.Data) != ""
		if vnicHasPublicIp {
			hasPublicIp = true
		}

		// Subnet-level gate: a subnet that prohibits internet ingress blocks
		// reachability regardless of the security groups.
		subnetProhibits := false
		subnet := vnic.GetSubnet()
		if subnet.Error != nil {
			return nil, subnet.Error
		}
		if subnet.Data != nil {
			p := subnet.Data.GetProhibitInternetIngress()
			if p.Error != nil {
				return nil, p.Error
			}
			subnetProhibits = p.Data
		}

		// Network security groups attached to this VNIC.
		nsgs := vnic.GetSecurityGroups()
		if nsgs.Error != nil {
			return nil, nsgs.Error
		}
		vnicOpenRules, nsgAllows, err := ociCollectOpenNsgRules(nsgs.Data)
		if err != nil {
			return nil, err
		}
		openRules = append(openRules, vnicOpenRules...)

		// securityGroupAllowsIngress reflects the NSG verdict alone, so it is not
		// muddied by the subnet gate (a user seeing it false can conclude no NSG
		// admits traffic). The subnet's internet-ingress policy is applied only to
		// internetReachable.
		if nsgAllows {
			securityGroupAllowsIngress = true
		}
		if !subnetProhibits && nsgAllows {
			if vnicHasPublicIp {
				internetReachable = true
			}
		}
	}

	res, err := CreateResource(i.MqlRuntime, "oci.network.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData("oci.compute.instance/" + id.Data + "/exposure"),
		"internetReachable":          llx.BoolData(internetReachable),
		"hasPublicIp":                llx.BoolData(hasPublicIp),
		"securityGroupAllowsIngress": llx.BoolData(securityGroupAllowsIngress),
		"openIngressRules":           llx.ArrayData(openRules, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciNetworkExposure), nil
}

// exposure breaks down whether the load balancer is reachable from the internet:
// it is not private, carries a public IP, has at least one listener accepting
// traffic, and its attached network security groups admit inbound from any
// address (or it has no NSG attached, OCI's default open posture). NSGs are
// inspected so a public load balancer fronted by a restrictive NSG is not
// reported reachable, matching the instance path.
func (l *mqlOciLoadBalancerLoadBalancer) exposure() (*mqlOciNetworkExposure, error) {
	id := l.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	isPrivate := l.GetIsPrivate()
	if isPrivate.Error != nil {
		return nil, isPrivate.Error
	}

	ips := l.GetIpAddresses()
	if ips.Error != nil {
		return nil, ips.Error
	}
	hasPublicIp := false
	if !isPrivate.Data {
		for _, e := range ips.Data {
			if d, ok := e.(map[string]any); ok {
				if pub, _ := d["isPublic"].(bool); pub {
					hasPublicIp = true
					break
				}
			}
		}
	}

	listeners := l.GetListeners()
	if listeners.Error != nil {
		return nil, listeners.Error
	}
	hasListener := len(listeners.Data) > 0

	nsgs := l.GetSecurityGroups()
	if nsgs.Error != nil {
		return nil, nsgs.Error
	}
	openRules, securityGroupAllowsIngress, err := ociCollectOpenNsgRules(nsgs.Data)
	if err != nil {
		return nil, err
	}

	internetReachable := hasPublicIp && hasListener && securityGroupAllowsIngress

	res, err := CreateResource(l.MqlRuntime, "oci.network.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData("oci.loadBalancer.loadBalancer/" + id.Data + "/exposure"),
		"internetReachable":          llx.BoolData(internetReachable),
		"hasPublicIp":                llx.BoolData(hasPublicIp),
		"securityGroupAllowsIngress": llx.BoolData(securityGroupAllowsIngress),
		"openIngressRules":           llx.ArrayData(openRules, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciNetworkExposure), nil
}

// internetReachable reports whether the autonomous database listener is
// reachable from the public internet: it has a public endpoint (no private
// endpoint) and either access control is disabled or its allow-list admits any
// address. mTLS may still be required to connect, but the endpoint is reachable.
func (a *mqlOciDatabaseAutonomousDatabase) internetReachable() (bool, error) {
	privateEndpoint := a.GetPrivateEndpointIp()
	if privateEndpoint.Error != nil {
		return false, privateEndpoint.Error
	}
	// A private endpoint means the database is only reachable inside the VCN.
	if strings.TrimSpace(privateEndpoint.Data) != "" {
		return false, nil
	}

	accessControl := a.GetIsAccessControlEnabled()
	if accessControl.Error != nil {
		return false, accessControl.Error
	}
	if !accessControl.Data {
		// Public endpoint with no network ACL — reachable from anywhere.
		return true, nil
	}

	whitelist := a.GetWhitelistedIps()
	if whitelist.Error != nil {
		return false, whitelist.Error
	}
	return ociWhitelistOpensInternet(whitelist.Data), nil
}

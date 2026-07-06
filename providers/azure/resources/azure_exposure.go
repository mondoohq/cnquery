// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// --- Pure helpers (table-tested in azure_exposure_test.go) ---

// internetOpenSourcePrefixes is the set of NSG source address prefixes that
// represent "any internet source". An inbound Allow rule whose source matches
// one of these exposes the destination to the public internet.
//   - "*"            wildcard, any source
//   - "0.0.0.0/0"    all IPv4
//   - "::/0"         all IPv6
//   - "internet"     the Azure "Internet" service tag (everything outside the VNet)
func isInternetOpenSourcePrefix(prefix string) bool {
	p := strings.ToLower(strings.TrimSpace(prefix))
	switch p {
	case "*", "0.0.0.0/0", "::/0", "internet":
		return true
	default:
		return false
	}
}

// securityRuleAllowsInternetIngress reports whether a single NSG security rule
// opens inbound traffic to the public internet. A rule qualifies when it is an
// inbound Allow rule whose source (single prefix or any entry in the prefix
// list) is an internet-open source. Direction/access matching is
// case-insensitive to tolerate API casing variations.
func securityRuleAllowsInternetIngress(direction, access, sourcePrefix string, sourcePrefixes []string) bool {
	if !strings.EqualFold(strings.TrimSpace(direction), "Inbound") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(access), "Allow") {
		return false
	}
	if isInternetOpenSourcePrefix(sourcePrefix) {
		return true
	}
	for _, p := range sourcePrefixes {
		if isInternetOpenSourcePrefix(p) {
			return true
		}
	}
	return false
}

// publicNetworkAccessEnabled interprets the Azure `publicNetworkAccess` string,
// which is "Enabled"/"Disabled" on most resources (and may be empty when the
// API omits it). Empty is treated as enabled because Azure defaults public
// access on when the property is not explicitly set to "Disabled".
func publicNetworkAccessEnabled(value string) bool {
	return !strings.EqualFold(strings.TrimSpace(value), "Disabled")
}

// firewallRuleAllowsAnyInternet reports whether a database firewall rule (start
// IP / end IP range) opens the server to the public internet. A rule qualifies
// when it starts at 0.0.0.0 and extends to any address beyond it — this catches
// the full IPv4 span (0.0.0.0 -> 255.255.255.255) as well as wide partial
// ranges such as 0.0.0.0 -> 128.255.255.255.
//
// The special "allow all Azure services" rule (0.0.0.0 -> 0.0.0.0) is
// deliberately NOT treated as internet-open. That rule permits traffic only
// from Azure-internal service IPs, not from arbitrary public addresses, so
// counting it as internet-reachable would be a false positive for servers that
// have nothing but that rule enabled.
func firewallRuleAllowsAnyInternet(startIp, endIp string) bool {
	start := strings.TrimSpace(startIp)
	end := strings.TrimSpace(endIp)
	return start == "0.0.0.0" && end != "" && end != "0.0.0.0"
}

// databaseInternetReachable combines the publicNetworkAccess gate with the
// presence of at least one internet-opening firewall rule. A database is
// internet-reachable only when public access is enabled AND some firewall rule
// permits an internet-wide source range.
func databaseInternetReachable(publicNetworkAccess string, firewallRanges [][2]string) bool {
	if !publicNetworkAccessEnabled(publicNetworkAccess) {
		return false
	}
	for _, r := range firewallRanges {
		if firewallRuleAllowsAnyInternet(r[0], r[1]) {
			return true
		}
	}
	return false
}

// aksApiServerInternetReachable reports whether an AKS API server is reachable
// from the public internet. It is reachable only when the cluster is not a
// private cluster, public network access is not disabled, and no authorized-IP
// allowlist restricts API access. Any of those gates closes the exposure.
func aksApiServerInternetReachable(enablePrivateCluster bool, publicNetworkAccess string, authorizedIPRanges []string) bool {
	if enablePrivateCluster {
		return false
	}
	if !publicNetworkAccessEnabled(publicNetworkAccess) {
		return false
	}
	if len(authorizedIPRanges) > 0 {
		return false
	}
	return true
}

// --- Resolvers ---

// ruleSourceIsInternet reports whether an Azure effective-security-rule object
// has a source that covers the public internet. It reads the source both as the
// configured prefix(es) and as the expanded set (service tags resolved to
// CIDRs), matching any internet-open source.
func ruleSourceIsInternet(rule map[string]any) bool {
	if sp, ok := rule["sourceAddressPrefix"].(string); ok && isInternetOpenSourcePrefix(sp) {
		return true
	}
	for _, key := range []string{"sourceAddressPrefixes", "expandedSourceAddressPrefix"} {
		if arr, ok := rule[key].([]any); ok {
			for _, p := range arr {
				if s, ok := p.(string); ok && isInternetOpenSourcePrefix(s) {
					return true
				}
			}
		}
	}
	return false
}

// effectiveRuleAllowsInternetIngress reports whether a single Azure
// effective-security-rule object (the raw dict returned by a NIC's
// effectiveSecurityRules) is an inbound Allow rule open to any internet source.
func effectiveRuleAllowsInternetIngress(rule map[string]any) bool {
	direction, _ := rule["direction"].(string)
	if !strings.EqualFold(strings.TrimSpace(direction), "Inbound") {
		return false
	}
	access, _ := rule["access"].(string)
	if !strings.EqualFold(strings.TrimSpace(access), "Allow") {
		return false
	}
	return ruleSourceIsInternet(rule)
}

// ruleInt extracts an integer field from an effective-rule dict. JSON numbers
// decode as float64 through encoding/json, so both float64 and integer forms
// are accepted.
func ruleInt(rule map[string]any, key string) (int, bool) {
	switch v := rule[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}

// portInterval is an inclusive [lo, hi] port range.
type portInterval struct{ lo, hi int }

// rulePortIntervals reads a rule's destination port range(s) into intervals.
// "*", an empty value, or an absent value means all ports (0-65535). Both the
// single destinationPortRange and the destinationPortRanges list are read.
func rulePortIntervals(rule map[string]any) []portInterval {
	var out []portInterval
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || s == "*" {
			out = append(out, portInterval{0, 65535})
			return
		}
		if i := strings.IndexByte(s, '-'); i >= 0 {
			lo, err1 := strconv.Atoi(strings.TrimSpace(s[:i]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(s[i+1:]))
			if err1 == nil && err2 == nil {
				out = append(out, portInterval{lo, hi})
			}
			return
		}
		if n, err := strconv.Atoi(s); err == nil {
			out = append(out, portInterval{n, n})
		}
	}
	if sp, ok := rule["destinationPortRange"].(string); ok {
		add(sp)
	}
	if arr, ok := rule["destinationPortRanges"].([]any); ok {
		for _, p := range arr {
			if s, ok := p.(string); ok {
				add(s)
			}
		}
	}
	if len(out) == 0 {
		out = append(out, portInterval{0, 65535})
	}
	return out
}

// portsCover reports whether the deny intervals fully contain every allow
// interval (each allow interval must fall within a single deny interval).
func portsCover(deny, allow []portInterval) bool {
	for _, a := range allow {
		covered := false
		for _, d := range deny {
			if d.lo <= a.lo && a.hi <= d.hi {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

// protocolCovers reports whether a deny rule's protocol covers an allow rule's
// protocol. "*"/"Any"/empty on the deny side covers every protocol.
func protocolCovers(deny, allow string) bool {
	deny = strings.TrimSpace(deny)
	if deny == "" || deny == "*" || strings.EqualFold(deny, "Any") {
		return true
	}
	return strings.EqualFold(deny, strings.TrimSpace(allow))
}

// destAddressIsBroad reports whether a destination prefix covers all addresses.
func destAddressIsBroad(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "*" || s == "0.0.0.0/0" || s == "::/0"
}

// ruleDestPrefixes reads a rule's destination address prefix(es), reading both
// the single destinationAddressPrefix and the destinationAddressPrefixes list.
// An absent/empty destination means all addresses, represented as "*".
func ruleDestPrefixes(rule map[string]any) []string {
	var out []string
	if s, ok := rule["destinationAddressPrefix"].(string); ok && strings.TrimSpace(s) != "" {
		out = append(out, s)
	}
	if arr, ok := rule["destinationAddressPrefixes"].([]any); ok {
		for _, p := range arr {
			if s, ok := p.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out
}

// destCovers reports whether a deny rule's destination covers an allow rule's
// destination: every one of the allow rule's destinations must be an
// all-addresses prefix on the deny side or equal to one of the deny rule's
// destinations. Both the single and plural forms are read on each side. When a
// destination is not covered we conservatively report false, leaving the allow
// rule un-shadowed (a security audit should err toward reporting exposure
// rather than hiding it).
func destCovers(deny, allow map[string]any) bool {
	denyDests := ruleDestPrefixes(deny)
	covers := func(target string) bool {
		for _, d := range denyDests {
			if destAddressIsBroad(d) || strings.EqualFold(d, target) {
				return true
			}
		}
		return false
	}
	for _, a := range ruleDestPrefixes(allow) {
		if !covers(a) {
			return false
		}
	}
	return true
}

// denyDominatesAllow reports whether a higher-priority Deny rule fully shadows
// an Allow rule for internet ingress: it must cover the Allow's protocol,
// destination ports, and destination address.
func denyDominatesAllow(deny, allow map[string]any) bool {
	dproto, _ := deny["protocol"].(string)
	aproto, _ := allow["protocol"].(string)
	if !protocolCovers(dproto, aproto) {
		return false
	}
	if !portsCover(rulePortIntervals(deny), rulePortIntervals(allow)) {
		return false
	}
	return destCovers(deny, allow)
}

// nsgAllowsInternetIngress reports whether a single NSG's effective rules admit
// inbound traffic from the internet, honoring rule priority, and returns the
// surviving internet-open Allow rules. An inbound internet Allow rule survives
// when no higher-priority (lower-numbered) inbound internet Deny rule dominates
// it (same protocol, destination ports, and destination). When no internet
// source rule allows ingress the group admits nothing (Azure's default
// DenyAllInbound applies).
//
// Only rules whose source covers the internet are considered — for both Allow
// and Deny. A Deny whose source is a non-internet tag (VirtualNetwork,
// AzureLoadBalancer, a private CIDR) does not block internet-sourced traffic,
// so it correctly never shadows an internet Allow here.
func nsgAllowsInternetIngress(rules []map[string]any) (bool, []map[string]any) {
	type prioritized struct {
		rule  map[string]any
		prio  int
		allow bool
	}
	var internetRules []prioritized
	for _, r := range rules {
		dir, _ := r["direction"].(string)
		if !strings.EqualFold(strings.TrimSpace(dir), "Inbound") {
			continue
		}
		if !ruleSourceIsInternet(r) {
			continue
		}
		prio, ok := ruleInt(r, "priority")
		if !ok {
			prio = int(^uint(0) >> 1) // unknown priority sorts last
		}
		access, _ := r["access"].(string)
		internetRules = append(internetRules, prioritized{
			rule:  r,
			prio:  prio,
			allow: strings.EqualFold(strings.TrimSpace(access), "Allow"),
		})
	}
	sort.SliceStable(internetRules, func(i, j int) bool {
		return internetRules[i].prio < internetRules[j].prio
	})

	var open []map[string]any
	for i, a := range internetRules {
		if !a.allow {
			continue
		}
		shadowed := false
		for j := 0; j < i; j++ {
			d := internetRules[j]
			if d.allow {
				continue
			}
			if denyDominatesAllow(d.rule, a.rule) {
				shadowed = true
				break
			}
		}
		if !shadowed {
			open = append(open, a.rule)
		}
	}
	return len(open) > 0, open
}

// exposure builds the network-exposure summary for a VM from its already-cached
// public IPs and the effective security rules of its NICs.
//
// Inbound internet traffic must be admitted by every NSG in a NIC's effective
// chain (the subnet-level NSG and the NIC-level NSG are evaluated in sequence),
// so each NIC is evaluated per-NSG: the NIC admits internet ingress only when
// all of its effective NSGs do. The VM is exposed when any NIC admits it. NICs
// with no effective NSG (stopped/detached, or the API could not compute rules)
// contribute nothing. Resolving effective rules is a live Azure call per NIC;
// it is only paid when exposure is queried.
//
// Rule priority is honored: within an NSG a higher-priority (lower-numbered)
// Deny rule shadows a lower-priority Allow-from-internet rule when it covers the
// same protocol, destination ports, and destination (see nsgAllowsInternetIngress).
//
// Limitation: AVNM security-admin rules, which override NSGs tenant-wide, are
// not folded into this evaluation.
func (a *mqlAzureSubscriptionComputeServiceVm) exposure() (*mqlAzureSubscriptionNetworkServiceExposure, error) {
	publicIps := a.GetPublicIpAddresses()
	if publicIps.Error != nil {
		return nil, publicIps.Error
	}
	hasPublicIp := len(publicIps.Data) > 0

	nics := a.GetNetworkInterfaces()
	if nics.Error != nil {
		return nil, nics.Error
	}

	securityGroupAllowsIngress := false
	openRules := []any{}
	for _, n := range nics.Data {
		nic, ok := n.(*mqlAzureSubscriptionNetworkServiceInterface)
		if !ok {
			continue
		}
		groups, err := nic.effectiveNsgGroupsCached()
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 {
			continue
		}
		nicAdmits := true
		var nicRules []map[string]any
		for _, g := range groups {
			allows, surviving := nsgAllowsInternetIngress(g.rules)
			if !allows {
				nicAdmits = false
				break
			}
			nicRules = append(nicRules, surviving...)
		}
		if nicAdmits {
			securityGroupAllowsIngress = true
			for _, r := range nicRules {
				openRules = append(openRules, r)
			}
		}
	}

	internetReachable := hasPublicIp && securityGroupAllowsIngress

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData("azure.subscription.computeService.vm/" + a.Id.Data + "/exposure"),
		"internetReachable":          llx.BoolData(internetReachable),
		"hasPublicIp":                llx.BoolData(hasPublicIp),
		"securityGroupAllowsIngress": llx.BoolData(securityGroupAllowsIngress),
		"openIngressRules":           llx.ArrayData(openRules, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceExposure), nil
}

// sqlFirewallRanges collects (startIp, endIp) pairs from a list of MQL SQL
// firewall-rule resources, ignoring rules whose accessor lookups error.
func sqlFirewallRanges(rules []any) [][2]string {
	out := make([][2]string, 0, len(rules))
	for _, r := range rules {
		fr, ok := r.(*mqlAzureSubscriptionSqlServiceFirewallrule)
		if !ok {
			continue
		}
		out = append(out, [2]string{fr.GetStartIpAddress().Data, fr.GetEndIpAddress().Data})
	}
	return out
}

func (a *mqlAzureSubscriptionSqlServiceServer) internetReachable() (bool, error) {
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	rules := a.GetFirewallRules()
	if rules.Error != nil {
		return false, rules.Error
	}
	return databaseInternetReachable(pna.Data, sqlFirewallRanges(rules.Data)), nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceFlexibleServer) internetReachable() (bool, error) {
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	rules := a.GetFirewallRules()
	if rules.Error != nil {
		return false, rules.Error
	}
	return databaseInternetReachable(pna.Data, sqlFirewallRanges(rules.Data)), nil
}

func (a *mqlAzureSubscriptionPostgreSqlServiceServer) internetReachable() (bool, error) {
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	rules := a.GetFirewallRules()
	if rules.Error != nil {
		return false, rules.Error
	}
	return databaseInternetReachable(pna.Data, sqlFirewallRanges(rules.Data)), nil
}

func (a *mqlAzureSubscriptionMySqlServiceServer) internetReachable() (bool, error) {
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	rules := a.GetFirewallRules()
	if rules.Error != nil {
		return false, rules.Error
	}
	return databaseInternetReachable(pna.Data, sqlFirewallRanges(rules.Data)), nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) internetReachable() (bool, error) {
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	rules := a.GetFirewallRules()
	if rules.Error != nil {
		return false, rules.Error
	}
	return databaseInternetReachable(pna.Data, sqlFirewallRanges(rules.Data)), nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) internetReachable() (bool, error) {
	priv := a.GetEnablePrivateCluster()
	if priv.Error != nil {
		return false, priv.Error
	}
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	rangesVal := a.GetApiServerAuthorizedIPRanges()
	if rangesVal.Error != nil {
		return false, rangesVal.Error
	}
	ranges := []string{}
	for _, r := range rangesVal.Data {
		if s, ok := r.(string); ok {
			ranges = append(ranges, s)
		}
	}
	return aksApiServerInternetReachable(priv.Data, pna.Data, ranges), nil
}

// storageAccountIsPublic combines the three gates that must all be open for a
// storage account to allow anonymous public reads: public network access not
// disabled, the network rule set defaulting to Allow, and blob containers
// permitted to be made anonymously public.
func storageAccountIsPublic(publicNetworkAccess, networkRuleDefaultAction string, allowBlobPublicAccess bool) bool {
	if !publicNetworkAccessEnabled(publicNetworkAccess) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(networkRuleDefaultAction), "Allow") {
		return false
	}
	return allowBlobPublicAccess
}

func (a *mqlAzureSubscriptionStorageServiceAccount) isPublic() (bool, error) {
	pna := a.GetPublicNetworkAccess()
	if pna.Error != nil {
		return false, pna.Error
	}
	defaultAction := a.GetNetworkRuleDefaultAction()
	if defaultAction.Error != nil {
		return false, defaultAction.Error
	}
	allowBlobPublic := a.GetAllowBlobPublicAccess()
	if allowBlobPublic.Error != nil {
		return false, allowBlobPublic.Error
	}
	return storageAccountIsPublic(pna.Data, defaultAction.Data, allowBlobPublic.Data), nil
}

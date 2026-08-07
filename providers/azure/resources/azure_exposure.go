// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"math"
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// --- Pure helpers (table-tested in azure_exposure_test.go) ---

// isInternetOpenSourcePrefix reports whether a single NSG source address prefix
// represents "any internet source". An inbound Allow rule whose source matches
// one of these exposes the destination to the public internet.
//
// The named forms are Azure's own vocabulary:
//   - "*" / "any"  wildcard, any source
//   - "internet"   the Azure "Internet" service tag (everything outside the VNet)
//
// Everything else is parsed as a prefix rather than string-compared, so any
// zero-length prefix counts -- "0.0.0.0/0" and "::/0", but equally "0.0.0.0/0 "
// or an IPv6 form written out in full. A string compare recognized only the two
// canonical spellings, and a rule written any other way read as closed.
//
// Note this deliberately answers only for ONE prefix. A source list can cover
// the whole internet without any single entry doing so -- see
// prefixesCoverInternet.
func isInternetOpenSourcePrefix(prefix string) bool {
	p := strings.ToLower(strings.TrimSpace(prefix))
	switch p {
	case "*", "any", "internet":
		return true
	}
	parsed, err := netip.ParsePrefix(p)
	if err != nil {
		return false
	}
	return parsed.Bits() == 0
}

// prefixesCoverInternet reports whether a set of source prefixes together cover
// the whole public address space, even though no single entry does.
//
// The case that matters is the split-halves form -- ["0.0.0.0/1",
// "128.0.0.0/1"] -- which is all of IPv4 written as two prefixes. It is a
// common way to express "any" and it defeats a per-entry check: every entry
// looks like an ordinary CIDR, so an NSG opened this way read as closed and the
// resource behind it reported internetReachable: false. Any zero-length IPv6
// prefix is treated the same way.
func prefixesCoverInternet(prefixes []string) bool {
	var ranges []ipRange
	for _, p := range prefixes {
		if isInternetOpenSourcePrefix(p) {
			return true
		}
		parsed, err := netip.ParsePrefix(strings.TrimSpace(p))
		if err != nil || !parsed.Addr().Is4() {
			continue
		}
		lo := ipv4ToUint(parsed.Masked().Addr())
		size := uint64(1) << (32 - parsed.Bits())
		ranges = append(ranges, ipRange{lo: uint64(lo), hi: uint64(lo) + size - 1})
	}
	return rangesCoverAllIPv4(ranges)
}

type ipRange struct{ lo, hi uint64 }

func ipv4ToUint(addr netip.Addr) uint32 {
	b := addr.As4()
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// rangesCoverAllIPv4 merges the ranges and reports whether they span the entire
// IPv4 address space.
func rangesCoverAllIPv4(ranges []ipRange) bool {
	if len(ranges) == 0 {
		return false
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].lo < ranges[j].lo })
	if ranges[0].lo != 0 {
		return false
	}
	reached := ranges[0].hi
	for _, r := range ranges[1:] {
		// a gap means the space is not fully covered
		if r.lo > reached+1 {
			return false
		}
		if r.hi > reached {
			reached = r.hi
		}
	}
	return reached >= math.MaxUint32
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
	return prefixesCoverInternet(sourcePrefixes)
}

// publicNetworkAccessEnabled interprets the Azure `publicNetworkAccess` string,
// which is "Enabled"/"Disabled" on most resources (and may be empty when the
// API omits it). Empty is treated as enabled because Azure defaults public
// access on when the property is not explicitly set to "Disabled".
func publicNetworkAccessEnabled(value string) bool {
	return !strings.EqualFold(strings.TrimSpace(value), "Disabled")
}

// firewallRuleAllowsAnyInternet reports whether a database firewall rule (start
// IP / end IP range) opens the server to the public internet.
//
// The rule is judged on how much of the address space it actually admits, not
// on the text of its endpoints. Anchoring on the literal string "0.0.0.0" got
// this wrong in both directions: 1.0.0.0 -> 255.255.255.255 and
// 0.0.0.1 -> 255.255.255.255 are the whole routable internet and read as
// closed, while 0.0.0.0 -> 0.0.0.1 is two addresses and read as open.
//
// A rule counts as internet-open when it spans at least half of IPv4. That
// keeps the documented wide-partial case (0.0.0.0 -> 128.255.255.255) and the
// off-by-one variants above, and excludes ordinary allowlists.
//
// The special "allow all Azure services" rule (0.0.0.0 -> 0.0.0.0) is NOT
// internet-open: it permits traffic only from Azure-internal service IPs, not
// from arbitrary public addresses. The span test excludes it on its own, since
// it admits a single address.
func firewallRuleAllowsAnyInternet(startIp, endIp string) bool {
	start, err := netip.ParseAddr(strings.TrimSpace(startIp))
	if err != nil || !start.Is4() {
		return false
	}
	end, err := netip.ParseAddr(strings.TrimSpace(endIp))
	if err != nil || !end.Is4() {
		return false
	}
	lo, hi := ipv4ToUint(start), ipv4ToUint(end)
	if hi < lo {
		return false
	}
	const halfOfIPv4 = uint64(1) << 31
	return uint64(hi)-uint64(lo)+1 >= halfOfIPv4
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
// from the public internet. It is reachable when the cluster is not private,
// public network access is not disabled, and no authorized-IP allowlist
// meaningfully restricts API access.
//
// An allowlist only counts as a restriction when it actually restricts
// something. Treating any non-empty list as a restriction meant a cluster
// allowlisted to 0.0.0.0/0 -- what several Terraform modules emit when the
// field cannot be left empty -- reported its API server as unreachable while it
// was open to the world. The same applies to a list written as the two IPv4
// halves.
func aksApiServerInternetReachable(enablePrivateCluster bool, publicNetworkAccess string, authorizedIPRanges []string) bool {
	if enablePrivateCluster {
		return false
	}
	if !publicNetworkAccessEnabled(publicNetworkAccess) {
		return false
	}
	if len(authorizedIPRanges) == 0 {
		return true
	}
	return prefixesCoverInternet(authorizedIPRanges)
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
		arr, ok := rule[key].([]any)
		if !ok {
			continue
		}
		prefixes := make([]string, 0, len(arr))
		for _, p := range arr {
			if s, ok := p.(string); ok {
				prefixes = append(prefixes, s)
			}
		}
		// judged as a set, not entry by entry: a source list can cover the
		// whole internet without any single entry doing so
		if prefixesCoverInternet(prefixes) {
			return true
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
		// An empty string means "this form is not in use" (Azure populates
		// destinationPortRanges and leaves the singular empty, or vice versa),
		// not "all ports". Let the len(out) == 0 fallback below decide, so an
		// empty singular can't widen a deny rule to cover every port.
		if s == "" {
			return
		}
		if s == "*" {
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
// protocol. "All"/"*"/"Any"/empty on the deny side covers every protocol.
//
// The effective-rules API emits "All" (armnetwork.EffectiveSecurityRuleProtocol
// is one of All, Tcp, Udp); the "*" and "Any" spellings come from the raw NSG
// rule model. Both are accepted so the helper works against either shape.
func protocolCovers(deny, allow string) bool {
	deny = strings.TrimSpace(deny)
	if deny == "" || deny == "*" || strings.EqualFold(deny, "Any") || strings.EqualFold(deny, "All") {
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
// all of its effective NSGs do. The VM is exposed when any NIC admits it. A NIC
// that Azure reports as having no NSG at all admits every inbound flow and so
// counts as exposed; a NIC whose rules could not be computed (stopped VM,
// access denied) is skipped and clears securityGroupsEvaluated instead, so a
// "closed" verdict is never inferred from a failed lookup. Resolving effective
// rules is a live Azure call per NIC; it is only paid when exposure is queried.
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
	// Every NIC must be evaluated authoritatively before a "closed" verdict
	// means anything; one degraded fetch makes the whole verdict provisional.
	allEvaluated := true
	openRules := []any{}
	for _, n := range nics.Data {
		nic, ok := n.(*mqlAzureSubscriptionNetworkServiceInterface)
		if !ok {
			continue
		}
		groups, evaluated, err := nic.effectiveNsgGroupsCached()
		if err != nil {
			return nil, err
		}
		if !evaluated {
			allEvaluated = false
			continue
		}
		if len(groups) == 0 {
			// Azure answered, and the NIC has no NSG on itself or its subnet.
			// Nothing filters inbound traffic, so this is the most exposed
			// configuration there is — not an absence of evidence.
			securityGroupAllowsIngress = true
			openRules = append(openRules, map[string]any{
				"name":                     "NoNetworkSecurityGroup",
				"access":                   "Allow",
				"direction":                "Inbound",
				"protocol":                 "All",
				"sourceAddressPrefix":      "*",
				"destinationAddressPrefix": "*",
				"destinationPortRange":     "0-65535",
			})
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

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionNetworkServiceExposure, map[string]*llx.RawData{
		"__id":                       llx.StringData("azure.subscription.computeService.vm/" + a.Id.Data + "/exposure"),
		"internetReachable":          llx.BoolData(internetReachable),
		"hasPublicIp":                llx.BoolData(hasPublicIp),
		"securityGroupAllowsIngress": llx.BoolData(securityGroupAllowsIngress),
		"securityGroupsEvaluated":    llx.BoolData(allEvaluated),
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

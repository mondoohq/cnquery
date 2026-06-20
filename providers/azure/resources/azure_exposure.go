// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
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
// IP / end IP range) opens the server to the entire public internet — a rule
// whose range spans the whole IPv4 space (0.0.0.0 -> 255.255.255.255).
//
// The special "allow all Azure services" rule (0.0.0.0 -> 0.0.0.0) is
// deliberately NOT treated as internet-open. That rule permits traffic only
// from Azure-internal service IPs, not from arbitrary public addresses, so
// counting it as internet-reachable would be a false positive for servers that
// have nothing but that rule enabled.
func firewallRuleAllowsAnyInternet(startIp, endIp string) bool {
	return strings.TrimSpace(startIp) == "0.0.0.0" && strings.TrimSpace(endIp) == "255.255.255.255"
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

func (a *mqlAzureSubscriptionNetworkServiceExposure) id() (string, error) {
	return a.Id.Data, nil
}

// exposure builds the network-exposure summary for a VM from its already-cached
// public IPs and the security rules of NSGs attached to its NICs. No new API
// calls are made beyond the existing publicIpAddresses()/networkInterfaces()
// accessors, both of which are cached on the VM resource.
//
// Limitation: NSG rule priorities are not evaluated. Azure applies rules in
// priority order (lowest number wins), so a higher-priority Deny rule can
// shadow a lower-priority Allow-from-internet rule. This breakdown flags any
// matching Allow rule as opening ingress, so securityGroupAllowsIngress may
// report true even when a higher-priority Deny actually blocks the traffic.
func (a *mqlAzureSubscriptionComputeServiceVm) exposure() (*mqlAzureSubscriptionNetworkServiceExposure, error) {
	publicIps := a.GetPublicIpAddresses()
	if publicIps.Error != nil {
		return nil, publicIps.Error
	}
	hasPublicIp := len(publicIps.Data) > 0

	openRules := []any{}
	nics := a.GetNetworkInterfaces()
	if nics.Error != nil {
		return nil, nics.Error
	}
	for _, n := range nics.Data {
		nic, ok := n.(*mqlAzureSubscriptionNetworkServiceInterface)
		if !ok {
			continue
		}
		nsgVal := nic.GetNetworkSecurityGroup()
		if nsgVal.Error != nil {
			// Propagate rather than swallow: a failed NSG load (e.g. a
			// permissions error) would otherwise make the VM look like it has no
			// open ingress rules — a false negative for internet exposure.
			return nil, nsgVal.Error
		}
		if nsgVal.Data == nil {
			// A NIC legitimately may have no NSG attached.
			continue
		}
		rules := nsgVal.Data.GetSecurityRules()
		if rules.Error != nil {
			return nil, rules.Error
		}
		for _, r := range rules.Data {
			rule, ok := r.(*mqlAzureSubscriptionNetworkServiceSecurityrule)
			if !ok {
				continue
			}
			direction := rule.GetDirection()
			access := rule.GetAccess()
			sourcePrefix := rule.GetSourceAddressPrefix()
			sourcePrefixesVal := rule.GetSourceAddressPrefixes()
			sourcePrefixes := []string{}
			for _, p := range sourcePrefixesVal.Data {
				if s, ok := p.(string); ok {
					sourcePrefixes = append(sourcePrefixes, s)
				}
			}
			if securityRuleAllowsInternetIngress(direction.Data, access.Data, sourcePrefix.Data, sourcePrefixes) {
				openRules = append(openRules, rule)
			}
		}
	}

	securityGroupAllowsIngress := len(openRules) > 0
	internetReachable := hasPublicIp && securityGroupAllowsIngress

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData("azure.subscription.computeService.vm/" + a.Id.Data + "/exposure"),
		"id":                         llx.StringData(a.Id.Data + "/exposure"),
		"internetReachable":          llx.BoolData(internetReachable),
		"hasPublicIp":                llx.BoolData(hasPublicIp),
		"securityGroupAllowsIngress": llx.BoolData(securityGroupAllowsIngress),
		"openIngressRules":           llx.ArrayData(openRules, "azure.subscription.networkService.securityrule"),
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

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// firewallRuleAllowsPublicIngress reports whether a Proxmox firewall rule
// permits inbound traffic from any source.
//
// Proxmox VE is on-prem virtualization, so "internet exposure" only maps
// meaningfully at the firewall-rule level: a rule that accepts inbound
// traffic from an unrestricted source opens whatever it guards (a VM,
// container, node, or the whole cluster) to every network the host bridge
// can reach. The predicate is true only when ALL of the following hold:
//
//   - the rule is enabled,
//   - it is an inbound rule (type == "in"); empty type also defaults to
//     inbound in PVE, but only ACCEPT verbs below qualify,
//   - the action accepts traffic (ACCEPT, case-insensitive),
//   - the source is unrestricted (see sourceIsAnywhere).
//
// Outbound rules, DROP/REJECT rules, disabled rules, group references, and
// rules scoped to a specific CIDR/IP/alias/IPset all return false. This is a
// pure function over already-cached rule fields — it makes no API calls.
func firewallRuleAllowsPublicIngress(ruleType, action, source string, enable bool) bool {
	if !enable {
		return false
	}

	t := strings.ToLower(strings.TrimSpace(ruleType))
	// Only inbound rules expose a target. PVE treats an empty type as an
	// inbound match, so accept "" as well; "out" and "group" do not. This
	// intentionally over-reports: a misconfigured rule with type="" and
	// action=ACCEPT is flagged as internet-exposed. For an exposure check that
	// is the safer direction — over-reporting beats silently missing a real
	// exposure. (See the "empty type defaults inbound" case in the test.)
	if t != "in" && t != "" {
		return false
	}

	if !strings.EqualFold(strings.TrimSpace(action), "ACCEPT") {
		return false
	}

	return sourceIsAnywhere(source)
}

// sourceIsAnywhere reports whether a Proxmox firewall rule source matches
// every address. Proxmox accepts the source in several forms:
//
//   - empty string — PVE treats an absent source as "any",
//   - the all-IPv4 route 0.0.0.0/0 (any prefix length that is effectively a
//     /0, e.g. "0.0.0.0/0"),
//   - the all-IPv6 route ::/0,
//   - a bare "0.0.0.0" or "::" with no prefix.
//
// A reference to a named alias or IPset (e.g. "+myset" or "dc/myalias") or
// any concrete CIDR/host is NOT considered "anywhere" — its breadth depends
// on data this function deliberately does not resolve, so it returns false
// to avoid over-reporting. Multiple comma-separated sources count as
// "anywhere" if ANY entry is an all-addresses match.
func sourceIsAnywhere(source string) bool {
	s := strings.TrimSpace(source)
	if s == "" {
		return true
	}

	for _, part := range strings.Split(s, ",") {
		if entryIsAnywhere(part) {
			return true
		}
	}
	return false
}

func entryIsAnywhere(entry string) bool {
	e := strings.TrimSpace(entry)
	if e == "" {
		// An empty entry within a list behaves like an absent source.
		return true
	}

	// A leading '+' denotes an IPset reference and '!' a negation; neither is
	// an unconditional all-addresses match.
	if strings.HasPrefix(e, "+") || strings.HasPrefix(e, "!") {
		return false
	}

	addr := e
	prefix := ""
	if idx := strings.IndexByte(e, '/'); idx >= 0 {
		addr = strings.TrimSpace(e[:idx])
		prefix = strings.TrimSpace(e[idx+1:])
	}

	isAllZeroV4 := addr == "0.0.0.0"
	isAllZeroV6 := addr == "::"
	if !isAllZeroV4 && !isAllZeroV6 {
		return false
	}

	// No prefix on the all-zero address means the host itself, which in
	// practice only matches when used as a wildcard; treat 0.0.0.0 / :: as
	// "anywhere".
	if prefix == "" {
		return true
	}

	// With a prefix, only a /0 covers every address.
	return prefix == "0"
}

// allowsPublicIngress is the generated resolver for
// proxmox.firewall.rule.allowsPublicIngress. It reads only fields already
// populated on the rule resource (type, action, source, enable) — no API
// call, no new permissions.
func (r *mqlProxmoxFirewallRule) allowsPublicIngress() (bool, error) {
	return firewallRuleAllowsPublicIngress(
		r.Type.Data,
		r.Action.Data,
		r.Source.Data,
		r.Enable.Data,
	), nil
}

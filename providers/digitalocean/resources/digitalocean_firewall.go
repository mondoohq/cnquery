// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
)

// mqlDigitaloceanFirewallIngressRuleInternal caches the source target IDs
// of an ingress rule so the typed source* accessors can resolve them
// against the parent's resource indexes.
type mqlDigitaloceanFirewallIngressRuleInternal struct {
	sourceDropletIDs       []any
	sourceLoadBalancerUIDs []any
	sourceKubernetesIDs    []any
}

// mqlDigitaloceanFirewallEgressRuleInternal mirrors the ingress variant
// for an egress rule's destination targets.
type mqlDigitaloceanFirewallEgressRuleInternal struct {
	destinationDropletIDs       []any
	destinationLoadBalancerUIDs []any
	destinationKubernetesIDs    []any
}

// ruleString returns the string value stored under key in a firewall-rule
// dict, or the empty string when absent.
func ruleString(rule map[string]any, key string) string {
	if v, ok := rule[key].(string); ok {
		return v
	}
	return ""
}

// ruleSlice returns the list stored under key in a firewall-rule dict, or
// an empty slice when absent.
func ruleSlice(rule map[string]any, key string) []any {
	if v, ok := rule[key].([]any); ok {
		return v
	}
	return []any{}
}

// openToInternet reports whether any of the given source/destination CIDRs
// admit traffic from (or to) every address — the IPv4 or IPv6 "everything"
// range.
func openToInternet(addresses []any) bool {
	for _, a := range addresses {
		if s, ok := a.(string); ok && (s == "0.0.0.0/0" || s == "::/0") {
			return true
		}
	}
	return false
}

func (r *mqlDigitaloceanFirewall) ingressRules() ([]any, error) {
	rules := r.GetInboundRules()
	if rules.Error != nil {
		return nil, rules.Error
	}
	out := make([]any, 0, len(rules.Data))
	for i, raw := range rules.Data {
		rule, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		addresses := ruleSlice(rule, "sourceAddresses")

		res, err := CreateResource(r.MqlRuntime, "digitalocean.firewall.ingressRule", map[string]*llx.RawData{
			"__id":            llx.StringData(fmt.Sprintf("%s/inbound/%d", r.Id.Data, i)),
			"protocol":        llx.StringData(ruleString(rule, "protocol")),
			"ports":           llx.StringData(ruleString(rule, "ports")),
			"openToInternet":  llx.BoolData(openToInternet(addresses)),
			"sourceAddresses": llx.ArrayData(addresses, "\x02"),
			"sourceTags":      llx.ArrayData(ruleSlice(rule, "sourceTags"), "\x02"),
		})
		if err != nil {
			return nil, err
		}
		mqlRule := res.(*mqlDigitaloceanFirewallIngressRule)
		mqlRule.sourceDropletIDs = ruleSlice(rule, "sourceDropletIds")
		mqlRule.sourceLoadBalancerUIDs = ruleSlice(rule, "sourceLoadBalancerUids")
		mqlRule.sourceKubernetesIDs = ruleSlice(rule, "sourceKubernetesIds")
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDigitaloceanFirewall) egressRules() ([]any, error) {
	rules := r.GetOutboundRules()
	if rules.Error != nil {
		return nil, rules.Error
	}
	out := make([]any, 0, len(rules.Data))
	for i, raw := range rules.Data {
		rule, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		addresses := ruleSlice(rule, "destinationAddresses")

		res, err := CreateResource(r.MqlRuntime, "digitalocean.firewall.egressRule", map[string]*llx.RawData{
			"__id":                 llx.StringData(fmt.Sprintf("%s/outbound/%d", r.Id.Data, i)),
			"protocol":             llx.StringData(ruleString(rule, "protocol")),
			"ports":                llx.StringData(ruleString(rule, "ports")),
			"openToInternet":       llx.BoolData(openToInternet(addresses)),
			"destinationAddresses": llx.ArrayData(addresses, "\x02"),
			"destinationTags":      llx.ArrayData(ruleSlice(rule, "destinationTags"), "\x02"),
		})
		if err != nil {
			return nil, err
		}
		mqlRule := res.(*mqlDigitaloceanFirewallEgressRule)
		mqlRule.destinationDropletIDs = ruleSlice(rule, "destinationDropletIds")
		mqlRule.destinationLoadBalancerUIDs = ruleSlice(rule, "destinationLoadBalancerUids")
		mqlRule.destinationKubernetesIDs = ruleSlice(rule, "destinationKubernetesIds")
		out = append(out, res)
	}
	return out, nil
}

// --- inbound rule typed source refs ---

func (r *mqlDigitaloceanFirewallIngressRule) sourceDroplets() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.dropletByIDs(r.sourceDropletIDs)
}

func (r *mqlDigitaloceanFirewallIngressRule) sourceLoadBalancers() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.loadBalancerByUIDs(r.sourceLoadBalancerUIDs)
}

func (r *mqlDigitaloceanFirewallIngressRule) sourceKubernetesClusters() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.kubernetesClustersByIDs(r.sourceKubernetesIDs)
}

// --- outbound rule typed destination refs ---

func (r *mqlDigitaloceanFirewallEgressRule) destinationDroplets() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.dropletByIDs(r.destinationDropletIDs)
}

func (r *mqlDigitaloceanFirewallEgressRule) destinationLoadBalancers() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.loadBalancerByUIDs(r.destinationLoadBalancerUIDs)
}

func (r *mqlDigitaloceanFirewallEgressRule) destinationKubernetesClusters() ([]any, error) {
	parent, err := parentDigitalocean(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return parent.kubernetesClustersByIDs(r.destinationKubernetesIDs)
}

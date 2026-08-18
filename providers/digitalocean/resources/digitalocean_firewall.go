// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
)

// mqlDigitaloceanFirewallInternal caches the raw rule set and protected
// droplet IDs of a cloud firewall so the typed ingressRules(), egressRules(),
// and droplets() accessors can resolve them without a refetch.
type mqlDigitaloceanFirewallInternal struct {
	cacheInboundRules  []godo.InboundRule
	cacheOutboundRules []godo.OutboundRule
	cacheDropletIDs    []any
}

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

// openToInternet reports whether any of the given source/destination CIDRs
// admit traffic from (or to) every address — the IPv4 or IPv6 "everything"
// range.
func openToInternet(addresses []string) bool {
	for _, s := range addresses {
		if s == "0.0.0.0/0" || s == "::/0" {
			return true
		}
	}
	return false
}

func (r *mqlDigitaloceanFirewall) ingressRules() ([]any, error) {
	out := make([]any, 0, len(r.cacheInboundRules))
	for i, rule := range r.cacheInboundRules {
		var addresses, tags, loadBalancerUIDs, kubernetesIDs []string
		var dropletIDs []int
		if rule.Sources != nil {
			addresses = rule.Sources.Addresses
			tags = rule.Sources.Tags
			dropletIDs = rule.Sources.DropletIDs
			loadBalancerUIDs = rule.Sources.LoadBalancerUIDs
			kubernetesIDs = rule.Sources.KubernetesIDs
		}

		res, err := CreateResource(r.MqlRuntime, "digitalocean.firewall.ingressRule", map[string]*llx.RawData{
			"__id":            llx.StringData(fmt.Sprintf("%s/inbound/%d", r.Id.Data, i)),
			"protocol":        llx.StringData(rule.Protocol),
			"ports":           llx.StringData(rule.PortRange),
			"openToInternet":  llx.BoolData(openToInternet(addresses)),
			"sourceAddresses": llx.ArrayData(toStringSlice(addresses), "\x02"),
			"sourceTags":      llx.ArrayData(toStringSlice(tags), "\x02"),
		})
		if err != nil {
			return nil, err
		}
		mqlRule := res.(*mqlDigitaloceanFirewallIngressRule)
		mqlRule.sourceDropletIDs = toIntSlice(dropletIDs)
		mqlRule.sourceLoadBalancerUIDs = toStringSlice(loadBalancerUIDs)
		mqlRule.sourceKubernetesIDs = toStringSlice(kubernetesIDs)
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDigitaloceanFirewall) egressRules() ([]any, error) {
	out := make([]any, 0, len(r.cacheOutboundRules))
	for i, rule := range r.cacheOutboundRules {
		var addresses, tags, loadBalancerUIDs, kubernetesIDs []string
		var dropletIDs []int
		if rule.Destinations != nil {
			addresses = rule.Destinations.Addresses
			tags = rule.Destinations.Tags
			dropletIDs = rule.Destinations.DropletIDs
			loadBalancerUIDs = rule.Destinations.LoadBalancerUIDs
			kubernetesIDs = rule.Destinations.KubernetesIDs
		}

		res, err := CreateResource(r.MqlRuntime, "digitalocean.firewall.egressRule", map[string]*llx.RawData{
			"__id":                 llx.StringData(fmt.Sprintf("%s/outbound/%d", r.Id.Data, i)),
			"protocol":             llx.StringData(rule.Protocol),
			"ports":                llx.StringData(rule.PortRange),
			"openToInternet":       llx.BoolData(openToInternet(addresses)),
			"destinationAddresses": llx.ArrayData(toStringSlice(addresses), "\x02"),
			"destinationTags":      llx.ArrayData(toStringSlice(tags), "\x02"),
		})
		if err != nil {
			return nil, err
		}
		mqlRule := res.(*mqlDigitaloceanFirewallEgressRule)
		mqlRule.destinationDropletIDs = toIntSlice(dropletIDs)
		mqlRule.destinationLoadBalancerUIDs = toStringSlice(loadBalancerUIDs)
		mqlRule.destinationKubernetesIDs = toStringSlice(kubernetesIDs)
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

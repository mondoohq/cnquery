// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

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

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// nicsHavePublicIp reports whether any network interface dict carries a
// non-empty public IP.
func nicsHavePublicIp(nics []any) bool {
	for _, n := range nics {
		nic, ok := n.(map[string]any)
		if !ok {
			continue
		}
		if ip, ok := nic["publicIp"].(string); ok && ip != "" {
			return true
		}
	}
	return false
}

// securityRuleOpenToInternet reports whether a security group rule is an ingress
// rule whose remote CIDR admits any address (0.0.0.0/0 or ::/0).
func securityRuleOpenToInternet(direction, ipRange string) bool {
	if !strings.EqualFold(direction, "ingress") {
		return false
	}
	return ipRange == "0.0.0.0/0" || ipRange == "::/0"
}

// exposure breaks down whether the server is reachable from the internet: a
// network interface with a public IP combined with a security group ingress
// rule that admits any address.
func (s *mqlStackitServer) exposure() (*mqlStackitNetworkExposure, error) {
	id := s.GetId()
	if id.Error != nil {
		return nil, id.Error
	}

	nics := s.GetNics()
	if nics.Error != nil {
		return nil, nics.Error
	}
	hasPublicIp := nicsHavePublicIp(nics.Data)

	openRules := []any{}
	sgs := s.GetSecurityGroups()
	if sgs.Error != nil {
		return nil, sgs.Error
	}
	for _, g := range sgs.Data {
		sg, ok := g.(*mqlStackitSecurityGroup)
		if !ok {
			continue
		}
		rules := sg.GetRules()
		if rules.Error != nil {
			return nil, rules.Error
		}
		for _, r := range rules.Data {
			rule, ok := r.(*mqlStackitSecurityGroupRule)
			if !ok {
				continue
			}
			direction := rule.GetDirection()
			if direction.Error != nil {
				return nil, direction.Error
			}
			ipRange := rule.GetIpRange()
			if ipRange.Error != nil {
				return nil, ipRange.Error
			}
			if securityRuleOpenToInternet(direction.Data, ipRange.Data) {
				openRules = append(openRules, rule)
			}
		}
	}
	sgAllows := len(openRules) > 0
	internetReachable := hasPublicIp && sgAllows

	res, err := CreateResource(s.MqlRuntime, "stackit.network.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData("stackit.server/" + id.Data + "/exposure"),
		"internetReachable":          llx.BoolData(internetReachable),
		"hasPublicIp":                llx.BoolData(hasPublicIp),
		"securityGroupAllowsIngress": llx.BoolData(sgAllows),
		"openIngressRules":           llx.ArrayData(openRules, types.Resource("stackit.securityGroup.rule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitNetworkExposure), nil
}

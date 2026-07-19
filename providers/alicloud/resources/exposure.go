// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// internetExposed reports whether the ECS instance has inbound reachability from
// the public internet, before security-group rules are applied. An instance is
// exposed when it carries a directly-assigned public IP address or an associated
// elastic IP address.
func (i *mqlAlicloudEcsInstance) internetExposed() (bool, error) {
	if len(i.PublicIpAddresses.Data) > 0 {
		return true, nil
	}
	if strings.TrimSpace(i.EipAddress.Data) != "" {
		return true, nil
	}
	return false, nil
}

// internetFacing reports whether the CLB instance serves traffic from the public
// internet, which is the case when its address type is internet.
func (l *mqlAlicloudSlbLoadBalancer) internetFacing() (bool, error) {
	return strings.EqualFold(strings.TrimSpace(l.AddressType.Data), "internet"), nil
}

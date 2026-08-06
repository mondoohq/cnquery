// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// internetGatewayIdPrefix identifies an internet gateway in a route's
// polymorphic gatewayId. Egress-only internet gateways use "eigw-" and are
// deliberately not matched: they carry outbound IPv6 traffic only, so a subnet
// whose default route points at one is not reachable from the internet.
const internetGatewayIdPrefix = "igw-"

// routeIsDefault reports whether a route's destination matches every address,
// for either address family. A default route is what carries traffic with no
// more specific match, so it is the one that decides internet reachability.
func routeIsDefault(ipv4Cidr, ipv6Cidr string) bool {
	return ipv4Cidr == "0.0.0.0/0" || ipv6Cidr == "::/0"
}

// hasInternetGatewayPrefix reports whether a route's polymorphic gatewayId names
// an internet gateway. An egress-only internet gateway ("eigw-") does not match:
// its ID does not carry the "igw-" prefix, and it only carries outbound traffic.
func hasInternetGatewayPrefix(gatewayId string) bool {
	return strings.HasPrefix(gatewayId, internetGatewayIdPrefix)
}

// blockModeStopsIngress reports whether a subnet's internet gateway block mode
// prevents inbound internet traffic. Both block-bidirectional and block-ingress
// stop traffic arriving from an internet gateway; only "off" (and an empty
// value, for a subnet the API reported no mode for) leaves it flowing.
func blockModeStopsIngress(mode string) bool {
	return mode == "block-bidirectional" || mode == "block-ingress"
}

// isPublic reports whether the subnet is a public subnet: one whose effective
// route table sends a default route to an internet gateway.
//
// The route table comes from routeTable(), which already resolves the subnet's
// explicit association and falls back to the VPC main route table, so this
// matches what AWS actually applies to traffic. A blackholed route is skipped:
// it matches traffic and then discards it, which is the opposite of reachable.
func (a *mqlAwsVpcSubnet) isPublic() (bool, error) {
	blockMode := a.GetInternetGatewayBlockMode()
	if blockMode.Error != nil {
		return false, blockMode.Error
	}
	if blockModeStopsIngress(blockMode.Data) {
		return false, nil
	}

	routeTable := a.GetRouteTable()
	if routeTable.Error != nil {
		return false, routeTable.Error
	}
	if routeTable.Data == nil {
		return false, nil
	}

	routes := routeTable.Data.GetRouteEntries()
	if routes.Error != nil {
		return false, routes.Error
	}
	for _, r := range routes.Data {
		route, ok := r.(*mqlAwsVpcRoutetableRoute)
		if !ok {
			continue
		}
		state := route.GetState()
		if state.Error != nil {
			return false, state.Error
		}
		if !strings.EqualFold(state.Data, "active") {
			continue
		}

		ipv4 := route.GetDestinationCidrBlock()
		if ipv4.Error != nil {
			return false, ipv4.Error
		}
		ipv6 := route.GetDestinationIpv6CidrBlock()
		if ipv6.Error != nil {
			return false, ipv6.Error
		}
		if !routeIsDefault(ipv4.Data, ipv6.Data) {
			continue
		}

		gatewayId := route.GetGatewayId()
		if gatewayId.Error != nil {
			return false, gatewayId.Error
		}
		if hasInternetGatewayPrefix(gatewayId.Data) {
			return true, nil
		}
	}
	return false, nil
}

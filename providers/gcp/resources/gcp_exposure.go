// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// openCIDRs are the source ranges that mean "the entire internet".
var openCIDRs = map[string]struct{}{
	"0.0.0.0/0": {},
	"::/0":      {},
}

// isOpenCIDR reports whether a CIDR string represents the whole internet
// (IPv4 0.0.0.0/0 or IPv6 ::/0). Surrounding whitespace is tolerated.
func isOpenCIDR(cidr string) bool {
	_, ok := openCIDRs[strings.TrimSpace(cidr)]
	return ok
}

// gkeControlPlaneInternetReachable derives whether a GKE cluster's control
// plane (Kubernetes API server) is reachable from the public internet.
//
// It is reachable when the public IP endpoint is enabled AND either:
//   - master authorized networks is NOT enforced (any source IP may connect), or
//   - the authorized-networks allowlist itself contains an open CIDR
//     (0.0.0.0/0 or ::/0), which whitelists the entire internet.
//
// When the public endpoint is disabled the control plane is private and never
// internet-reachable, regardless of the authorized-networks configuration.
func gkeControlPlaneInternetReachable(publicEndpointEnabled, authorizedNetworksEnforced bool, authorizedCIDRs []string) bool {
	if !publicEndpointEnabled {
		return false
	}
	if !authorizedNetworksEnforced {
		return true
	}
	for _, c := range authorizedCIDRs {
		if isOpenCIDR(c) {
			return true
		}
	}
	return false
}

func (g *mqlGcpProjectGkeServiceCluster) controlPlaneInternetReachable() (bool, error) {
	if g.ControlPlanePublicEndpointEnabled.Error != nil {
		return false, g.ControlPlanePublicEndpointEnabled.Error
	}
	if g.MasterAuthorizedNetworksAllowed.Error != nil {
		return false, g.MasterAuthorizedNetworksAllowed.Error
	}
	if g.MasterAuthorizedNetworksCidrs.Error != nil {
		return false, g.MasterAuthorizedNetworksCidrs.Error
	}

	cidrs := make([]string, 0, len(g.MasterAuthorizedNetworksCidrs.Data))
	for _, raw := range g.MasterAuthorizedNetworksCidrs.Data {
		if s, ok := raw.(string); ok {
			cidrs = append(cidrs, s)
		}
	}

	return gkeControlPlaneInternetReachable(
		g.ControlPlanePublicEndpointEnabled.Data,
		g.MasterAuthorizedNetworksAllowed.Data,
		cidrs,
	), nil
}

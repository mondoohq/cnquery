// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

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

// firewallRuleOpenIngress reports whether a firewall rule admits inbound traffic
// from any address — an enabled INGRESS rule whose source ranges include
// 0.0.0.0/0 or ::/0.
func firewallRuleOpenIngress(isAllow bool, direction string, disabled bool, sourceRanges []any) bool {
	// A GCP VPC firewall rule is exclusively an allow rule or a deny rule. Only
	// allow rules can open ingress; a broad-source INGRESS deny rule (a common
	// "block all" pattern) must not be counted as reachable exposure.
	if !isAllow || disabled || !strings.EqualFold(direction, "INGRESS") {
		return false
	}
	for _, s := range sourceRanges {
		if cidr, ok := s.(string); ok && (cidr == "0.0.0.0/0" || cidr == "::/0") {
			return true
		}
	}
	return false
}

// networkNameFromUrl returns the trailing network name from a GCP network URL or
// partial reference, so full URLs and short names compare equal.
func networkNameFromUrl(url string) string {
	if i := strings.LastIndex(url, "/networks/"); i >= 0 {
		return url[i+len("/networks/"):]
	}
	if i := strings.LastIndex(url, "/"); i >= 0 {
		return url[i+1:]
	}
	return url
}

// firewallTargetsInstance reports whether a firewall rule's targeting applies to
// an instance. A rule with no target tags and no target service accounts applies
// to every instance in its network; otherwise it applies only when a target tag
// or target service account matches the instance.
func firewallTargetsInstance(targetTags, targetServiceAccounts []any, instanceTags, instanceServiceAccounts map[string]bool) bool {
	if len(targetTags) == 0 && len(targetServiceAccounts) == 0 {
		return true
	}
	for _, t := range targetTags {
		if tag, ok := t.(string); ok && instanceTags[tag] {
			return true
		}
	}
	for _, sa := range targetServiceAccounts {
		if email, ok := sa.(string); ok && instanceServiceAccounts[email] {
			return true
		}
	}
	return false
}

func anyStringSet(items []any) map[string]bool {
	set := map[string]bool{}
	for _, i := range items {
		if s, ok := i.(string); ok && s != "" {
			set[s] = true
		}
	}
	return set
}

func (g *mqlGcpProjectComputeServiceInstance) exposure() (*mqlGcpProjectComputeServiceInstanceExposure, error) {
	id := g.GetId()
	if id.Error != nil {
		return nil, id.Error
	}
	hasPublicIp := g.GetHasPublicIp()
	if hasPublicIp.Error != nil {
		return nil, hasPublicIp.Error
	}
	projectId := g.GetProjectId()
	if projectId.Error != nil {
		return nil, projectId.Error
	}

	// Networks the instance is attached to.
	nics := g.GetNetworkInterfaces()
	if nics.Error != nil {
		return nil, nics.Error
	}
	instanceNetworks := map[string]bool{}
	for _, n := range nics.Data {
		nic, ok := n.(map[string]any)
		if !ok {
			continue
		}
		if network, ok := nic["network"].(string); ok && network != "" {
			instanceNetworks[networkNameFromUrl(network)] = true
		}
	}

	tags := g.GetTags()
	if tags.Error != nil {
		return nil, tags.Error
	}
	instanceTags := anyStringSet(tags.Data)

	serviceAccounts := g.GetServiceAccounts()
	if serviceAccounts.Error != nil {
		return nil, serviceAccounts.Error
	}
	instanceServiceAccounts := map[string]bool{}
	for _, s := range serviceAccounts.Data {
		sa, ok := s.(*mqlGcpProjectComputeServiceServiceaccount)
		if !ok {
			continue
		}
		email := sa.GetEmail()
		if email.Error != nil {
			return nil, email.Error
		}
		if email.Data != "" {
			instanceServiceAccounts[email.Data] = true
		}
	}

	svc, err := NewResource(g.MqlRuntime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId.Data),
	})
	if err != nil {
		return nil, err
	}
	firewalls := svc.(*mqlGcpProjectComputeService).GetFirewalls()
	if firewalls.Error != nil {
		return nil, firewalls.Error
	}

	openFirewalls := []any{}
	for _, f := range firewalls.Data {
		fw, ok := f.(*mqlGcpProjectComputeServiceFirewall)
		if !ok {
			continue
		}
		direction := fw.GetDirection()
		if direction.Error != nil {
			return nil, direction.Error
		}
		disabled := fw.GetDisabled()
		if disabled.Error != nil {
			return nil, disabled.Error
		}
		sourceRanges := fw.GetSourceRanges()
		if sourceRanges.Error != nil {
			return nil, sourceRanges.Error
		}
		allowed := fw.GetAllowed()
		if allowed.Error != nil {
			return nil, allowed.Error
		}
		isAllow := len(allowed.Data) > 0
		if !firewallRuleOpenIngress(isAllow, direction.Data, disabled.Data, sourceRanges.Data) {
			continue
		}
		if !instanceNetworks[networkNameFromUrl(fw.cacheNetworkUrl)] {
			continue
		}
		targetTags := fw.GetTargetTags()
		if targetTags.Error != nil {
			return nil, targetTags.Error
		}
		targetServiceAccounts := fw.GetTargetServiceAccounts()
		if targetServiceAccounts.Error != nil {
			return nil, targetServiceAccounts.Error
		}
		if firewallTargetsInstance(targetTags.Data, targetServiceAccounts.Data, instanceTags, instanceServiceAccounts) {
			openFirewalls = append(openFirewalls, fw)
		}
	}

	firewallAllowsIngress := len(openFirewalls) > 0
	internetReachable := hasPublicIp.Data && firewallAllowsIngress

	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.instance.exposure", map[string]*llx.RawData{
		"__id":                  llx.StringData("gcp.project.computeService.instance/" + id.Data + "/exposure"),
		"internetReachable":     llx.BoolData(internetReachable),
		"hasPublicIp":           llx.BoolData(hasPublicIp.Data),
		"firewallAllowsIngress": llx.BoolData(firewallAllowsIngress),
		"openIngressFirewalls":  llx.ArrayData(openFirewalls, types.Resource("gcp.project.computeService.firewall")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceInstanceExposure), nil
}

// internetReachable reports whether the Cloud SQL instance is reachable from the
// internet: it has a public IP and an authorized network that admits any address
// (0.0.0.0/0). Reuses the existing publicIpEnabled and hasOpenAuthorizedNetworks
// signals.
func (g *mqlGcpProjectSqlServiceInstance) internetReachable() (bool, error) {
	public := g.GetPublicIpEnabled()
	if public.Error != nil {
		return false, public.Error
	}
	if !public.Data {
		return false, nil
	}
	settings := g.GetSettings()
	if settings.Error != nil {
		return false, settings.Error
	}
	if settings.Data == nil {
		return false, nil
	}
	ipConfig := settings.Data.GetIpConfiguration()
	if ipConfig.Error != nil {
		return false, ipConfig.Error
	}
	if ipConfig.Data == nil {
		return false, nil
	}
	open := ipConfig.Data.GetHasOpenAuthorizedNetworks()
	if open.Error != nil {
		return false, open.Error
	}
	return open.Data, nil
}

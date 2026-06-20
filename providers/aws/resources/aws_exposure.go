// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// viewerPolicyEnforcesHttps reports whether a CloudFront viewer protocol policy
// requires HTTPS. "allow-all" permits plaintext HTTP; "https-only" and
// "redirect-to-https" enforce TLS. An empty policy is treated as not enforcing.
func viewerPolicyEnforcesHttps(policy string) bool {
	return policy != "" && !strings.EqualFold(policy, "allow-all")
}

// listenerProtocolIsPlaintext reports whether a load balancer listener protocol
// carries traffic without transport encryption.
func listenerProtocolIsPlaintext(protocol string) bool {
	switch strings.ToUpper(protocol) {
	case "HTTP", "TCP", "UDP", "TCP_UDP":
		return true
	default:
		return false
	}
}

// listenerDescriptionProtocol extracts the protocol from a load balancer
// listener-description dict, handling both shapes: ALB/NLB listeners (from
// DescribeListeners) carry "Protocol" at the top level, while classic ELB
// listener descriptions nest it under "Listener".
func listenerDescriptionProtocol(desc map[string]any) string {
	if p, ok := desc["Protocol"].(string); ok && p != "" {
		return p
	}
	if listener, ok := desc["Listener"].(map[string]any); ok {
		if p, ok := listener["Protocol"].(string); ok {
			return p
		}
	}
	return ""
}

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

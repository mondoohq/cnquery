// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "strings"

// classifyNamingContexts separates domain naming contexts from application
// partitions so callers can search the current domain, forest root domain, and
// child domains without accidentally treating DNS application partitions as
// domain NCs.
func classifyNamingContexts(namingContexts []string) (string, string, []string) {
	var domainDnsZonesDN string
	var forestDnsZonesDN string

	domainNCs := make([]string, 0, len(namingContexts))
	seen := make(map[string]struct{}, len(namingContexts))

	for _, nc := range namingContexts {
		upper := strings.ToUpper(strings.TrimSpace(nc))
		switch {
		case strings.HasPrefix(upper, "DC=DOMAINDNSZONES,"):
			if domainDnsZonesDN == "" {
				domainDnsZonesDN = nc
			}
		case strings.HasPrefix(upper, "DC=FORESTDNSZONES,"):
			if forestDnsZonesDN == "" {
				forestDnsZonesDN = nc
			}
		case strings.HasPrefix(upper, "DC="):
			if _, ok := seen[upper]; ok {
				continue
			}
			seen[upper] = struct{}{}
			domainNCs = append(domainNCs, nc)
		}
	}

	return domainDnsZonesDN, forestDnsZonesDN, domainNCs
}

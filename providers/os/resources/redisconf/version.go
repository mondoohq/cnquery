// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package redisconf

import (
	"regexp"
	"strings"
)

// reVersion matches the banner redis-server and valkey-server print, for
// example "Redis server v=7.4.1 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64"
// and "Valkey server v=8.0.1 ...". Capturing the product alongside the
// version is what lets a binary probe report the flavor without a second
// call.
var reVersion = regexp.MustCompile(`(?i)\b(redis|valkey)\s+server\s+v=(\d+\.\d+(?:\.\d+)?)`)

// ParseVersion extracts the product and version from a server binary's
// --version output. Both are empty when the output carries no banner.
func ParseVersion(output string) (product string, version string) {
	m := reVersion.FindStringSubmatch(output)
	if m == nil {
		return "", ""
	}
	if strings.EqualFold(m[1], "valkey") {
		return "valkey", m[2]
	}
	return "redis", m[2]
}

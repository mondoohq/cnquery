// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDropletHasPublicAddress(t *testing.T) {
	// Regression: prior implementation only considered PublicIpv4, so
	// an IPv6-only droplet would always report missingFirewall=false
	// regardless of actual coverage.
	cases := []struct {
		name, v4, v6 string
		want         bool
	}{
		{"no public addresses", "", "", false},
		{"public ipv4 only", "203.0.113.10", "", true},
		{"public ipv6 only", "", "2001:db8::1", true},
		{"both ipv4 and ipv6", "203.0.113.10", "2001:db8::1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, dropletHasPublicAddress(tc.v4, tc.v6))
		})
	}
}

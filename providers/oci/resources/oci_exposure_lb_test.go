// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOciIpIsPublic(t *testing.T) {
	yes, no := true, false

	tests := []struct {
		name        string
		isPublic    *bool
		lbIsPrivate bool
		want        bool
	}{
		// An explicit flag always wins.
		{"explicitly public", &yes, false, true},
		{"explicitly private", &no, false, false},
		{"explicitly public on a private balancer", &yes, true, true},

		// The case that matters: isPublic is optional on both load balancer
		// SDK models. Absent on a balancer that is not private means public.
		// Defaulting to false here reported a genuinely internet-facing
		// balancer as unreachable, so an "no load balancer may be exposed"
		// policy passed with nothing to fail on.
		{"absent on a public balancer", nil, false, true},
		{"absent on a private balancer", nil, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ociIpIsPublic(tt.isPublic, tt.lbIsPrivate))
		})
	}
}

func TestOciLoadBalancerHasPublicIp(t *testing.T) {
	ip := func(addr string, isPublic any) map[string]any {
		return map[string]any{"ipAddress": addr, "isPublic": isPublic}
	}

	tests := []struct {
		name      string
		ips       []any
		isPrivate bool
		want      bool
	}{
		{"no addresses at all", nil, false, false},
		{"one public address", []any{ip("203.0.113.10", true)}, false, true},
		{"only private addresses", []any{ip("10.0.0.5", false)}, false, false},
		{"a public address among private ones", []any{
			ip("10.0.0.5", false), ip("203.0.113.10", true),
		}, false, true},

		// A private balancer's addresses are internal whatever the individual
		// flags say, so the balancer flag short-circuits.
		{"private balancer with a public-looking address", []any{
			ip("203.0.113.10", true),
		}, true, false},

		// The regression: the network load balancer marshalled the SDK slice
		// straight to JSON, so an optional isPublic arrived as null. Reading
		// that back as "not public" cleared the balancer entirely.
		{"missing isPublic on a public balancer", []any{
			map[string]any{"ipAddress": "203.0.113.10"},
		}, false, true},
		{"null isPublic on a public balancer", []any{ip("203.0.113.10", nil)}, false, true},
		{"null isPublic on a private balancer", []any{ip("10.0.0.5", nil)}, true, false},

		// Malformed entries must not panic or be mistaken for a verdict.
		{"non-map entry is skipped", []any{"203.0.113.10"}, false, false},
		{"non-map entry alongside a public one", []any{
			"garbage", ip("203.0.113.10", true),
		}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ociLoadBalancerHasPublicIp(tt.ips, tt.isPrivate))
		})
	}
}

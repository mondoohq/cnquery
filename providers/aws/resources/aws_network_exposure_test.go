// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

func TestSecurityGroupCount(t *testing.T) {
	assert.Equal(t, 0, securityGroupCount(nil), "a nil TValue means no groups")
	assert.Equal(t, 0, securityGroupCount(&plugin.TValue[[]any]{}))
	assert.Equal(t, 0, securityGroupCount(&plugin.TValue[[]any]{Data: []any{}}))
	assert.Equal(t, 2, securityGroupCount(&plugin.TValue[[]any]{Data: []any{1, 2}}))
}

// TestInternetReachableSemantics documents the rule buildNetworkExposure
// applies. It is expressed against the same inputs rather than constructing a
// resource, because CreateResource needs a runtime.
//
// The third case is the regression: an internet-facing NLB has no security
// groups (they cannot be attached after creation, and Gateway Load Balancers
// never support them), so requiring an open security-group rule reported every
// such load balancer as unreachable.
func TestInternetReachableSemantics(t *testing.T) {
	reachable := func(publiclyAccessible bool, sgCount int, sgAllows bool) bool {
		if sgCount > 0 {
			return publiclyAccessible && sgAllows
		}
		return publiclyAccessible
	}

	tests := []struct {
		name               string
		publiclyAccessible bool
		sgCount            int
		sgAllows           bool
		want               bool
	}{
		{"private resource is never reachable", false, 1, true, false},
		{"public resource with an open group is reachable", true, 1, true, true},
		{"public NLB with no security groups is reachable", true, 0, false, true},
		{"public resource whose groups are all closed is not reachable", true, 2, false, false},
		{"private resource with no groups is not reachable", false, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, reachable(tt.publiclyAccessible, tt.sgCount, tt.sgAllows))
		})
	}
}

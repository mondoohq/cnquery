// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCidrIsPublic(t *testing.T) {
	tests := []struct {
		cidr string
		want bool
	}{
		// all-addresses
		{"0.0.0.0/0", true},
		{"::/0", true},
		// broad public blocks that the literal-/0 check used to miss
		{"0.0.0.0/1", true},
		{"128.0.0.0/1", true},
		{"13.0.0.0/8", true},
		// private / reserved ranges are not internet exposure
		{"10.0.0.0/8", false},
		{"172.16.0.0/12", false},
		{"192.168.0.0/16", false},
		{"127.0.0.0/8", false},
		{"169.254.0.0/16", false},
		{"100.64.0.0/10", false},
		{"fc00::/7", false},
		// specific or narrow ranges are scoped, not "the internet"
		{"203.0.113.5/32", false},
		{"52.94.0.0/16", false},
		{"2600:1f00::/24", true},
		{"2001:db8::/48", false},
		// malformed / empty
		{"", false},
		{"not-a-cidr", false},
		{"0.0.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			assert.Equal(t, tt.want, cidrIsPublic(tt.cidr))
		})
	}
}

func TestAnyCidrPublic(t *testing.T) {
	assert.True(t, anyCidrPublic([]any{"10.0.0.0/8", "0.0.0.0/1"}))
	assert.False(t, anyCidrPublic([]any{"10.0.0.0/8", "192.168.1.0/24"}))
	assert.False(t, anyCidrPublic([]any{}))
	assert.False(t, anyCidrPublic([]any{42, "10.0.0.0/8"}))
}

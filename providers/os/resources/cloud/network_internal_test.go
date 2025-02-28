// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBroadcastAddressFrom(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected string
	}{
		{"Valid /24 subnet", "192.168.1.0/24", "192.168.1.255"},
		{"Valid /30 subnet", "192.168.1.0/30", "192.168.1.3"},
		{"Valid /16 subnet", "10.0.0.0/16", "10.0.255.255"},
		{"Valid /8 subnet", "172.0.0.0/8", "172.255.255.255"},
		{"Single IP /32", "192.168.1.1/32", "192.168.1.1"},
		{"Full range /0", "0.0.0.0/0", "255.255.255.255"},
		{"Invalid CIDR", "invalid", ""},
		{"IPv6 CIDR", "2001:db8::/32", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := broadcastAddressFrom(tt.cidr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

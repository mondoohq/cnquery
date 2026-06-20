// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirewallRuleOpenToInternet(t *testing.T) {
	tests := []struct {
		name string
		rule map[string]any
		want bool
	}{
		{"inbound any ipv4", map[string]any{"direction": "in", "sourceIps": []any{"0.0.0.0/0"}}, true},
		{"inbound any ipv6", map[string]any{"direction": "in", "sourceIps": []any{"::/0"}}, true},
		{"inbound mixed with any", map[string]any{"direction": "in", "sourceIps": []any{"10.0.0.0/8", "0.0.0.0/0"}}, true},
		{"inbound scoped source", map[string]any{"direction": "in", "sourceIps": []any{"203.0.113.0/24"}}, false},
		{"outbound any", map[string]any{"direction": "out", "sourceIps": []any{"0.0.0.0/0"}}, false},
		{"no sources", map[string]any{"direction": "in"}, false},
		{"empty", map[string]any{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, firewallRuleOpenToInternet(tt.rule))
		})
	}
}

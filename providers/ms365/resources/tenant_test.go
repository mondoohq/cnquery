// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnabledServicePlanServices(t *testing.T) {
	plan := func(service, status string) map[string]any {
		return map[string]any{"service": service, "capabilityStatus": status}
	}

	tests := []struct {
		name string
		in   []any
		want []string
	}{
		{"nil input", nil, []string{}},
		{"empty input", []any{}, []string{}},
		{
			name: "only enabled plans are returned, sorted",
			in: []any{
				plan("exchange", "Enabled"),
				plan("AADPremiumService", "Enabled"),
			},
			want: []string{"AADPremiumService", "exchange"},
		},
		{
			name: "non-enabled statuses are dropped",
			in: []any{
				plan("exchange", "Enabled"),
				plan("MicrosoftOffice", "Deleted"),
				plan("SharePoint", "Suspended"),
			},
			want: []string{"exchange"},
		},
		{
			name: "duplicate services collapse to one",
			in: []any{
				plan("exchange", "Enabled"),
				plan("exchange", "Enabled"),
			},
			want: []string{"exchange"},
		},
		{
			name: "status match is case-insensitive",
			in: []any{
				plan("exchange", "enabled"),
				plan("AADPremiumService", "ENABLED"),
			},
			want: []string{"AADPremiumService", "exchange"},
		},
		{
			name: "entries with empty service names are skipped",
			in: []any{
				plan("", "Enabled"),
				plan("exchange", "Enabled"),
			},
			want: []string{"exchange"},
		},
		{
			name: "non-map and malformed entries are skipped",
			in: []any{
				"not-a-map",
				map[string]any{"service": 7, "capabilityStatus": "Enabled"},
				map[string]any{"service": "exchange"}, // missing status
				plan("exchange", "Enabled"),
			},
			want: []string{"exchange"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, enabledServicePlanServices(tc.in))
		})
	}
}

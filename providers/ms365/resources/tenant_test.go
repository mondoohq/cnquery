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

func TestSumPaidLicenses(t *testing.T) {
	sub := func(sku, status string, trial bool, seats int32) companySubscription {
		return companySubscription{
			SkuPartNumber: ptr(sku),
			Status:        ptr(status),
			IsTrial:       ptr(trial),
			TotalLicenses: ptr(seats),
		}
	}

	tests := []struct {
		name string
		in   []companySubscription
		want int64
	}{
		{
			name: "free/self-service plans do not inflate the total",
			// Reproduces the observed 1,010,091: two sentinel free SKUs plus
			// 91 real paid seats. Only the 91 should count.
			in: []companySubscription{
				sub("RIGHTSMANAGEMENT_ADHOC", "Enabled", false, 1000000),
				sub("FLOW_FREE", "Enabled", false, 10000),
				sub("ENTERPRISEPREMIUM", "Enabled", false, 91),
			},
			want: 91,
		},
		{
			name: "trials are excluded",
			in: []companySubscription{
				sub("ENTERPRISEPREMIUM", "Enabled", false, 100),
				sub("SPE_E3", "Enabled", true, 50),
			},
			want: 100,
		},
		{
			name: "Warning (grace period) counts, Suspended/Deleted/LockedOut do not",
			in: []companySubscription{
				sub("ENTERPRISEPREMIUM", "Enabled", false, 100),
				sub("SPE_E3", "Warning", false, 40),
				sub("SPB", "Suspended", false, 30),
				sub("EMS", "Deleted", false, 20),
				sub("SPE_E5", "LockedOut", false, 10),
			},
			want: 140,
		},
		{
			name: "nil pointers are tolerated",
			in: []companySubscription{
				{TotalLicenses: ptr(int32(25))},                                   // nil status/trial/sku
				{SkuPartNumber: ptr("ENTERPRISEPREMIUM"), Status: ptr("Enabled")}, // nil seats
			},
			want: 25,
		},
		{
			name: "empty input is zero",
			in:   nil,
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sumPaidLicenses(tc.in))
		})
	}
}

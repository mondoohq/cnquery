// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLIPMappingID(t *testing.T) {
	// A Cloud SQL instance can expose more than one IP of the same Type
	// (e.g., two PRIVATE IPs across distinct VPC networks). The address
	// itself disambiguates them.
	cases := []struct {
		name string
		want string
		got  string
	}{
		{
			name: "primary",
			want: "inst1/ipAddresses/PRIMARY/34.10.20.30",
			got:  sqlIPMappingID("inst1", "PRIMARY", "34.10.20.30"),
		},
		{
			name: "first private",
			want: "inst1/ipAddresses/PRIVATE/10.0.0.5",
			got:  sqlIPMappingID("inst1", "PRIVATE", "10.0.0.5"),
		},
		{
			name: "second private on different VPC",
			want: "inst1/ipAddresses/PRIVATE/10.1.0.5",
			got:  sqlIPMappingID("inst1", "PRIVATE", "10.1.0.5"),
		},
	}
	seen := make(map[string]string, len(cases))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
			if prev, exists := seen[tc.got]; exists {
				t.Fatalf("id collision: %q produced by both %q and %q", tc.got, prev, tc.name)
			}
			seen[tc.got] = tc.name
		})
	}
}

func TestSQLDenyMaintenancePeriodID(t *testing.T) {
	// Two periods can begin on the same date (e.g., recurring annual
	// freezes) but end on different ones; both must participate in the id.
	cases := []struct {
		name string
		want string
		got  string
	}{
		{
			name: "single period",
			want: "inst1/settings/denyMaintenancePeriod/2026-01-01/2026-01-07",
			got:  sqlDenyMaintenancePeriodID("inst1", "2026-01-01", "2026-01-07"),
		},
		{
			name: "same start, different end",
			want: "inst1/settings/denyMaintenancePeriod/2026-01-01/2026-01-14",
			got:  sqlDenyMaintenancePeriodID("inst1", "2026-01-01", "2026-01-14"),
		},
		{
			name: "different start, same end",
			want: "inst1/settings/denyMaintenancePeriod/2026-02-01/2026-01-14",
			got:  sqlDenyMaintenancePeriodID("inst1", "2026-02-01", "2026-01-14"),
		},
	}
	seen := make(map[string]string, len(cases))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
			if prev, exists := seen[tc.got]; exists {
				t.Fatalf("id collision: %q produced by both %q and %q", tc.got, prev, tc.name)
			}
			seen[tc.got] = tc.name
		})
	}
}

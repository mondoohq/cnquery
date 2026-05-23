// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"
)

// TestDaysBetween pins the rounding direction of the shared helper
// behind mqlProxmoxCertificate.daysUntilExpiry. The test calls the
// production helper directly so a drift in the formula is caught
// here rather than going unnoticed.
func TestDaysBetween(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		to   time.Time
		want int64
	}{
		{"exactly-30-days", now.Add(30 * 24 * time.Hour), 30},
		{"29-days-23-hours", now.Add(29*24*time.Hour + 23*time.Hour), 29},
		{"29-days-25-hours", now.Add(29*24*time.Hour + 25*time.Hour), 30},
		{"5-hours-from-expiry-rounds-to-zero", now.Add(5 * time.Hour), 0},
		{"already-expired", now.Add(-72 * time.Hour), -3},
		{"expired-by-an-hour", now.Add(-1 * time.Hour), 0},
		{"expired-by-25-hours", now.Add(-25 * time.Hour), -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := daysBetween(now, tt.to)
			if got != tt.want {
				t.Errorf("daysBetween(now, %v) = %d, want %d", tt.to, got, tt.want)
			}
		})
	}
}

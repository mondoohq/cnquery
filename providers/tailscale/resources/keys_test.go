// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"
)

func TestTimeIsSet(t *testing.T) {
	set := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	zero := time.Time{}
	epoch := time.Unix(0, 0)

	tests := []struct {
		name string
		in   *time.Time
		want bool
	}{
		{name: "nil pointer", in: nil, want: false},
		{name: "zero time (0001-01-01)", in: &zero, want: false},
		{name: "real timestamp", in: &set, want: true},
		// Tailscale's "unset" is the Go zero time; a genuine Unix epoch 0 is
		// non-zero in Go terms and is treated as set.
		{name: "unix epoch 0", in: &epoch, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := timeIsSet(tc.in); got != tc.want {
				t.Fatalf("timeIsSet(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

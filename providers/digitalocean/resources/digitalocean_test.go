// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDoTime(t *testing.T) {
	// Regression: dict timestamps must render the zero time as "" rather
	// than a misleading "0001-01-01T00:00:00Z" when the API omits them.
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"zero value", time.Time{}, ""},
		{"unset pointer deref", time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), ""},
		{"real timestamp", time.Date(2026, time.June, 19, 18, 30, 0, 0, time.UTC), "2026-06-19T18:30:00Z"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatDoTime(tc.in))
		})
	}
}

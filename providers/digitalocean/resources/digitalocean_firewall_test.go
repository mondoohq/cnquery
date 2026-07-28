// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenToInternet(t *testing.T) {
	cases := []struct {
		name      string
		addresses []string
		want      bool
	}{
		{"ipv4 any", []string{"0.0.0.0/0"}, true},
		{"ipv6 any", []string{"::/0"}, true},
		{"any among specifics", []string{"10.0.0.0/8", "0.0.0.0/0"}, true},
		{"specific only", []string{"10.0.0.0/8", "203.0.113.5/32"}, false},
		{"ipv6 specific only", []string{"2001:db8::/32"}, false},
		{"host address is not a range", []string{"0.0.0.0"}, false},
		{"empty", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, openToInternet(c.addresses))
		})
	}
}

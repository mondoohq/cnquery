// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPqcKeyExchangeEnabled locks in the ML-KEM detection: a supported group is
// post-quantum iff its name contains "MLKEM" (case-insensitive), matching the
// Windows group strings such as secp256r1_mlkem768.
func TestPqcKeyExchangeEnabled(t *testing.T) {
	cases := []struct {
		name   string
		curves []string
		want   bool
	}{
		{
			name:   "mixed list with an ML-KEM group",
			curves: []string{"SecP256r1MLKEM768", "x25519"},
			want:   true,
		},
		{
			name:   "windows lowercase mlkem group string",
			curves: []string{"curve25519", "secp384r1_mlkem1024"},
			want:   true,
		},
		{
			name:   "only classical curves",
			curves: []string{"curve25519", "NistP256"},
			want:   false,
		},
		{
			name:   "empty list",
			curves: []string{},
			want:   false,
		},
		{
			name:   "nil list",
			curves: nil,
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, pqcKeyExchangeEnabled(c.curves))
		})
	}
}

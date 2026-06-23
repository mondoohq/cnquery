// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRhel67ReleaseRegex(t *testing.T) {
	tests := []struct {
		release string
		match   bool
	}{
		{"6", true},
		{"6.5", true},
		{"6Server", true},
		{"7", true},
		{"7.9", true},
		// the old `^[6|7].*$` alternation also matched a literal pipe, which
		// is what this regression guards against
		{"|", false},
		{"|garbage", false},
		{"8", false},
		{"5", false},
		{"", false},
	}

	for _, tt := range tests {
		assert.Equalf(t, tt.match, rhel67release.MatchString(tt.release),
			"rhel67release.MatchString(%q)", tt.release)
	}
}

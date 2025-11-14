// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHexFlags(t *testing.T) {
	tests := []struct {
		hexStr   string
		expected []string
	}{
		{"1", []string{"UP"}},
		{"2", []string{"BROADCAST"}},
		{"3", []string{"UP", "BROADCAST"}},
		{"8", []string{"LOOPBACK"}},
		{"10", []string{"POINTOPOINT"}},
		{"40", []string{"RUNNING"}},
		{"100", []string{"PROMISC"}},
		{"400", []string{"MASTER"}},
		{"1003", []string{"BROADCAST", "MULTICAST", "UP"}},
		{"8000", []string{"DYNAMIC"}},
		{"8001", []string{"UP", "DYNAMIC"}},
		{"FFFF", []string{"UP", "BROADCAST", "DEBUG", "LOOPBACK", "POINTOPOINT", "NOTRAILERS", "RUNNING", "NOARP", "PROMISC", "ALLMULTI", "MASTER", "SLAVE", "MULTICAST", "PORTSEL", "AUTOMEDIA", "DYNAMIC"}},
		{"0", []string{}},       // No flags set
		{"invalid", []string{}}, // Invalid hex input
	}

	for _, test := range tests {
		t.Run("hexStr="+test.hexStr, func(t *testing.T) {
			assert.ElementsMatch(t, test.expected, parseHexFlags(test.hexStr))
		})
	}
}

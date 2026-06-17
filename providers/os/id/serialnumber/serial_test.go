// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package serialnumber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidSerialNumber(t *testing.T) {
	tests := []struct {
		serial string
		valid  bool
	}{
		{"", false},
		{"System Serial Number", false},
		{"system serial number", false},
		{"Base Board Serial Number", false},
		{"Chassis Serial Number", false},
		{"To Be Filled By O.E.M.", false},
		{"Default string", false},
		{"Not Specified", false},
		{"None", false},
		{"0", false},
		{"00000000", false},
		{"OEM", false},
		// genuine-looking serials
		{"VMware-56 4d 1a 2b", true},
		{"PF2ABCDE", true},
		{"5CG1234ABC", true},
	}

	for _, tt := range tests {
		t.Run(tt.serial, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidSerialNumber(tt.serial))
		})
	}
}

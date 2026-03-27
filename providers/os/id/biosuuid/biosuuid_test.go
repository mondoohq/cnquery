// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package biosuuid

import (
	"testing"
)

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name  string
		uuid  string
		valid bool
	}{
		{"valid uuid", "64f118d3-0060-4a4c-bf1f-a11d655c4d6f", true},
		{"valid uuid uppercase", "16BD4D56-6B98-23F9-493C-F6B14E7CFC0B", true},
		{"empty string", "", false},
		{"all zeros", "00000000-0000-0000-0000-000000000000", false},
		{"not settable", "Not Settable", false},
		{"not available", "Not Available", false},
		{"not specified", "Not Specified", false},
		{"not present", "Not Present", false},
		{"none", "None", false},
		{"default string", "Default string", false},
		{"oem placeholder", "To Be Filled By O.E.M.", false},
		{"sentinel with whitespace", "  Not Settable  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUUID(tt.uuid)
			if got != tt.valid {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.uuid, got, tt.valid)
			}
		})
	}
}

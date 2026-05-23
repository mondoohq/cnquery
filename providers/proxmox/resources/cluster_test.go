// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestParseHAResourceSID(t *testing.T) {
	tests := []struct {
		name     string
		sid      string
		typ      string
		wantKind string
		wantID   int64
		wantOK   bool
	}{
		{"prefixed-vm", "vm:100", "", "vm", 100, true},
		{"prefixed-ct", "ct:200", "", "ct", 200, true},
		{"bare-numeric-falls-back-to-type", "300", "vm", "vm", 300, true},
		{"empty-input", "", "", "", 0, false},
		{"prefixed-but-non-numeric", "vm:not-a-number", "", "", 0, false},
		{"bare-non-numeric", "garbage", "vm", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, id, ok := parseHAResourceSID(tt.sid, tt.typ)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", kind, tt.wantKind)
			}
			if id != tt.wantID {
				t.Errorf("id = %d, want %d", id, tt.wantID)
			}
		})
	}
}

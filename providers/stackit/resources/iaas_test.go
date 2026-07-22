// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestProtocolLabel(t *testing.T) {
	cases := []struct {
		name      string
		protoName string
		number    int64
		hasNumber bool
		want      string
	}{
		{"name only", "tcp", 0, false, "tcp"},
		{"name wins over number", "udp", 17, true, "udp"},
		{"number fallback (GRE)", "", 47, true, "47"},
		{"number zero is still a protocol", "", 0, true, "0"},
		{"neither set", "", 0, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := protocolLabel(tc.protoName, tc.number, tc.hasNumber); got != tc.want {
				t.Fatalf("protocolLabel(%q, %d, %v) = %q, want %q",
					tc.protoName, tc.number, tc.hasNumber, got, tc.want)
			}
		})
	}
}

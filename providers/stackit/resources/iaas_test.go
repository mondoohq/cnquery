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

func TestClassifyVolumeSource(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	cases := []struct {
		name                              string
		sourceType                        string
		wantImage, wantSnapshot, wantBkup string
	}{
		{"image source", "image", id, "", ""},
		{"snapshot source", "snapshot", "", id, ""},
		{"backup source stays a backup, not a snapshot", "backup", "", "", id},
		{"volume clone maps to nothing", "volume", "", "", ""},
		{"unknown type maps to nothing", "something-new", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotImage, gotSnapshot, gotBackup := classifyVolumeSource(tc.sourceType, id)
			if gotImage != tc.wantImage || gotSnapshot != tc.wantSnapshot || gotBackup != tc.wantBkup {
				t.Fatalf("classifyVolumeSource(%q, id) = (%q, %q, %q), want (%q, %q, %q)",
					tc.sourceType, gotImage, gotSnapshot, gotBackup,
					tc.wantImage, tc.wantSnapshot, tc.wantBkup)
			}
		})
	}
}

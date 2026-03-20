// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import "testing"

func TestSetBoolOpt(t *testing.T) {
	tests := []struct {
		name     string
		value    []byte
		wantSet  bool
		wantVal  string
	}{
		{"binary true (\\x01)", []byte{0x01}, true, "true"},
		{"binary false (\\x00)", []byte{0x00}, false, ""},
		{"string true", []byte("true"), true, "true"},
		{"string false", []byte("false"), true, "true"}, // 'f' != 0, so treated as true
		{"nil", nil, false, ""},
		{"empty", []byte{}, false, ""},
		{"nonzero byte", []byte{0xff}, true, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := make(map[string]string)
			setBoolOpt(opts, "test-key", tt.value)
			val, ok := opts["test-key"]
			if ok != tt.wantSet {
				t.Errorf("setBoolOpt() set=%v, want %v", ok, tt.wantSet)
			}
			if val != tt.wantVal {
				t.Errorf("setBoolOpt() val=%q, want %q", val, tt.wantVal)
			}
		})
	}
}

func TestStrVal(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"path with null", []byte("/tmp/keytab\x00"), "/tmp/keytab"},
		{"clean string", []byte("alice@REALM"), "alice@REALM"},
		{"nil", nil, ""},
		{"empty", []byte{}, ""},
		{"only null", []byte{0x00}, ""},
		{"multi null", []byte("val\x00\x00"), "val"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strVal(tt.in); got != tt.want {
				t.Errorf("strVal(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSetStrOpt(t *testing.T) {
	opts := make(map[string]string)
	setStrOpt(opts, "k1", []byte("val\x00"))
	setStrOpt(opts, "k2", []byte{0x00})
	setStrOpt(opts, "k3", nil)
	if opts["k1"] != "val" {
		t.Errorf("k1 = %q, want %q", opts["k1"], "val")
	}
	if _, ok := opts["k2"]; ok {
		t.Error("k2 should not be set")
	}
	if _, ok := opts["k3"]; ok {
		t.Error("k3 should not be set")
	}
}
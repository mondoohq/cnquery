// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestIsSerialPortKey(t *testing.T) {
	tests := map[string]bool{
		"serial0":    true,
		"serial1":    true,
		"serial2":    true,
		"serial3":    true,
		"serial":     false, // missing slot index
		"serial4":    false, // PVE caps at serial3
		"serial10":   false, // two-digit slot is not valid
		"serialport": false, // not a real PVE config key
		"net0":       false,
		"":           false,
	}
	for key, want := range tests {
		t.Run(key+"=>"+boolstr(want), func(t *testing.T) {
			if got := isSerialPortKey(key); got != want {
				t.Errorf("isSerialPortKey(%q) = %v, want %v", key, got, want)
			}
		})
	}
}

func boolstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestParseRawLxcLines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "newline-delimited",
			in:   "lxc.apparmor.profile: unconfined\nlxc.cap.drop: sys_module",
			want: []string{
				"lxc.apparmor.profile: unconfined",
				"lxc.cap.drop: sys_module",
			},
		},
		{
			name: "literal-backslash-n (older API)",
			in:   "lxc.apparmor.profile: unconfined\\nlxc.cap.drop: mac_admin",
			want: []string{
				"lxc.apparmor.profile: unconfined",
				"lxc.cap.drop: mac_admin",
			},
		},
		{
			name: "blanks-and-comments-skipped",
			in:   "lxc.cap.drop: sys_module\n\n# disabled override\nlxc.cap.drop: mknod",
			want: []string{
				"lxc.cap.drop: sys_module",
				"lxc.cap.drop: mknod",
			},
		},
		{
			name: "empty-input",
			in:   "",
			want: nil,
		},
		{
			name: "single-line",
			in:   "lxc.apparmor.profile: unconfined",
			want: []string{"lxc.apparmor.profile: unconfined"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRawLxcLines(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRawLxcLines(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

// TestVMSensitiveFieldsAreNotInVmGoSource is a guard against accidentally
// extracting the cipassword value into a string field on the resource.
// If a future change ever maps cipassword -> a string accessor, this
// test will flag it during review even before any audit runs.
func TestVMCipasswordValueNeverExposed(t *testing.T) {
	src := mustReadFile(t, "vm.go")
	// The only valid mention of `cipassword` is the presence-only
	// accessor (cipasswordSet). Anything that reads `r.vmConfig["cipassword"]`
	// into a string-returning method is a regression.
	if containsSubstring(src, `r.cfgStr("cipassword")`) {
		t.Error("cipassword should never be read into a string field; only the presence flag is exposed")
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func containsSubstring(s, sub string) bool {
	return strings.Contains(s, sub)
}

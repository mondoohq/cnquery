// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestPciHostKey(t *testing.T) {
	for key, want := range map[string]bool{
		"hostpci0":  true,
		"hostpci15": true,
		"hostpci16": false, // PVE caps at 15
		"hostpci":   false,
		"hostpci99": false,
		"hostpcia":  false,
		"net0":      false,
		"":          false,
	} {
		t.Run(key+"="+boolstr(want), func(t *testing.T) {
			if got := pciHostKey(key); got != want {
				t.Errorf("pciHostKey(%q) = %v, want %v", key, got, want)
			}
		})
	}
}

func TestUsbVMKey(t *testing.T) {
	for key, want := range map[string]bool{
		"usb0":    true,
		"usb14":   true,
		"usb15":   false, // PVE caps at 14
		"usb":     false,
		"usbport": false,
		"":        false,
	} {
		t.Run(key+"="+boolstr(want), func(t *testing.T) {
			if got := usbVMKey(key); got != want {
				t.Errorf("usbVMKey(%q) = %v, want %v", key, got, want)
			}
		})
	}
}

func TestDevContainerKey(t *testing.T) {
	tests := map[string]bool{
		"dev0":    true,
		"dev255":  true,
		"dev256":  false, // PVE caps at 255
		"dev":     false,
		"dev01":   false, // canonical form is dev1
		"dev9999": false,
		"net0":    false,
	}
	for key, want := range tests {
		t.Run(key+"="+boolstr(want), func(t *testing.T) {
			if got := devContainerKey(key); got != want {
				t.Errorf("devContainerKey(%q) = %v, want %v", key, got, want)
			}
		})
	}
}

func TestParseHostPCIConfig_DirectAddress(t *testing.T) {
	args := parseHostPCIConfig("hostpci0", "0000:01:00.0,pcie=1,rombar=0,x-vga=1")
	if v := getStr(args, "address"); v != "0000:01:00.0" {
		t.Errorf("address = %q, want %q", v, "0000:01:00.0")
	}
	if v := getStr(args, "mapping"); v != "" {
		t.Errorf("mapping = %q, want empty", v)
	}
	if !getBool(args, "pciExpress") {
		t.Error("expected pciExpress=true")
	}
	if getBool(args, "romBar") {
		t.Error("expected romBar=false")
	}
	if !getBool(args, "xVga") {
		t.Error("expected xVga=true")
	}
}

func TestParseHostPCIConfig_Mapping(t *testing.T) {
	args := parseHostPCIConfig("hostpci2", "mapping=gpu0,mdev=nvidia-46")
	if v := getStr(args, "mapping"); v != "gpu0" {
		t.Errorf("mapping = %q, want %q", v, "gpu0")
	}
	if v := getStr(args, "address"); v != "" {
		t.Errorf("address = %q, want empty when mapping is set", v)
	}
	if v := getStr(args, "mdev"); v != "nvidia-46" {
		t.Errorf("mdev = %q, want %q", v, "nvidia-46")
	}
}

func TestParseHostPCIConfig_HostExplicitForm(t *testing.T) {
	args := parseHostPCIConfig("hostpci0", "host=0000:00:1f.6,pcie=1")
	if v := getStr(args, "address"); v != "0000:00:1f.6" {
		t.Errorf("address = %q, want %q", v, "0000:00:1f.6")
	}
	if !getBool(args, "pciExpress") {
		t.Error("expected pciExpress=true")
	}
}

func TestParseVMUsbConfig_HostVendorProduct(t *testing.T) {
	args := parseVMUsbConfig("usb0", "host=1234:5678,usb3=1")
	if v := getStr(args, "target"); v != "1234:5678" {
		t.Errorf("target = %q, want %q", v, "1234:5678")
	}
	if !getBool(args, "usb3") {
		t.Error("expected usb3=true")
	}
}

func TestParseVMUsbConfig_BusPath(t *testing.T) {
	args := parseVMUsbConfig("usb1", "host=1-2.3")
	if v := getStr(args, "target"); v != "1-2.3" {
		t.Errorf("target = %q, want %q", v, "1-2.3")
	}
}

func TestParseVMUsbConfig_SpiceSentinel(t *testing.T) {
	args := parseVMUsbConfig("usb2", "spice")
	if v := getStr(args, "target"); v != "spice" {
		t.Errorf("target = %q, want %q", v, "spice")
	}
}

func TestParseVMUsbConfig_HostDevicePath(t *testing.T) {
	// PVE accepts bare host device paths
	args := parseVMUsbConfig("usb3", "/dev/bus/usb/001/002")
	if v := getStr(args, "target"); v != "/dev/bus/usb/001/002" {
		t.Errorf("target = %q, want %q", v, "/dev/bus/usb/001/002")
	}
}

func TestParseContainerDeviceConfig_Full(t *testing.T) {
	args := parseContainerDeviceConfig("dev0", "/dev/dri/card0,uid=100000,gid=100044,mode=0660")
	if v := getStr(args, "path"); v != "/dev/dri/card0" {
		t.Errorf("path = %q, want %q", v, "/dev/dri/card0")
	}
	if v := getInt(args, "uid"); v != 100000 {
		t.Errorf("uid = %d, want 100000", v)
	}
	if v := getInt(args, "gid"); v != 100044 {
		t.Errorf("gid = %d, want 100044", v)
	}
	if v := getStr(args, "mode"); v != "0660" {
		t.Errorf("mode = %q, want %q", v, "0660")
	}
}

func TestParseContainerDeviceConfig_PathOnly(t *testing.T) {
	args := parseContainerDeviceConfig("dev1", "/dev/kvm")
	if v := getStr(args, "path"); v != "/dev/kvm" {
		t.Errorf("path = %q, want %q", v, "/dev/kvm")
	}
	if v := getInt(args, "uid"); v != 0 {
		t.Errorf("uid = %d, want 0 (unset)", v)
	}
	if v := getStr(args, "mode"); v != "" {
		t.Errorf("mode = %q, want empty (unset)", v)
	}
}

// TestVMPCISpotsAllSlots / TestVMUSBSpotsAllSlots are tight regression
// tests: they assert the resource list returned by hostpciN / usbN
// scanning covers every PVE-accepted slot index.
func TestPciHostKeyCoversFullRange(t *testing.T) {
	for i := 0; i <= 15; i++ {
		key := "hostpci" + itoa(i)
		if !pciHostKey(key) {
			t.Errorf("pciHostKey(%q) = false, expected true", key)
		}
	}
}

func TestUsbVMKeyCoversFullRange(t *testing.T) {
	for i := 0; i <= 14; i++ {
		key := "usb" + itoa(i)
		if !usbVMKey(key) {
			t.Errorf("usbVMKey(%q) = false, expected true", key)
		}
	}
}

func TestDevContainerKeyCoversFullRange(t *testing.T) {
	for _, i := range []int{0, 1, 9, 10, 99, 100, 255} {
		key := "dev" + itoa(i)
		if !devContainerKey(key) {
			t.Errorf("devContainerKey(%q) = false, expected true", key)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

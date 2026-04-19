// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFwupd(t *testing.T) {
	f, err := os.Open("testdata/fwupd.json")
	require.NoError(t, err)
	defer f.Close()

	devices, err := ParseFwupd(f)
	require.NoError(t, err)

	// First device has no Name, so it should be skipped
	assert.Len(t, devices, 3)

	// Intel Management Engine
	ime := devices[0]
	assert.Equal(t, "Intel Management Engine", ime.Name)
	assert.Equal(t, "5fed1486be004d67ea79838d2e83aaa11bb72645", ime.DeviceId)
	assert.Equal(t, "14.1.53.1649", ime.Version)
	assert.Equal(t, "Intel Corporation", ime.Vendor)
	assert.Equal(t, "MEI:0x8086", ime.VendorId)
	assert.Equal(t, "mei", ime.Plugin)
	assert.Equal(t, "org.uefi.capsule", ime.Protocol)
	assert.Equal(t, "intel-me", ime.VersionFormat)
	assert.False(t, ime.Updatable)
	assert.Len(t, ime.Guid, 2)
	assert.Contains(t, ime.Purl, "pkg:generic/intel-corporation/intel-management-engine@14.1.53.1649")

	// UEFI Device — has "updatable" flag
	uefi := devices[1]
	assert.Equal(t, "UEFI Device Firmware", uefi.Name)
	assert.True(t, uefi.Updatable)

	// Samsung SSD
	ssd := devices[2]
	assert.Equal(t, "Samsung SSD 980 PRO", ssd.Name)
	assert.Equal(t, "5B2QGXA7", ssd.Version)
	assert.Equal(t, "Samsung", ssd.Vendor)
}

func TestParseSystemProfiler(t *testing.T) {
	f, err := os.Open("testdata/system_profiler.json")
	require.NoError(t, err)
	defer f.Close()

	devices, err := ParseSystemProfiler(f)
	require.NoError(t, err)

	// Should have: Boot ROM, NVMe, USB Hub, USB YubiKey, Thunderbolt, Bluetooth, GPU
	assert.Len(t, devices, 7)

	// Boot ROM
	bootRom := devices[0]
	assert.Equal(t, "MacBook Pro Boot ROM", bootRom.Name)
	assert.Equal(t, "10151.140.19", bootRom.Version)
	assert.Equal(t, "Apple", bootRom.Vendor)
	assert.Equal(t, "SPHardwareDataType", bootRom.Plugin)

	// NVMe
	nvme := devices[1]
	assert.Equal(t, "APPLE SSD AP2048Z", nvme.Name)
	assert.Equal(t, "419.120.", nvme.Version)

	// USB Hub
	usbHub := devices[2]
	assert.Equal(t, "USB Hub", usbHub.Name)
	assert.Equal(t, "3.00", usbHub.Version)
	assert.Equal(t, "Apple Inc.", usbHub.Vendor)

	// USB YubiKey (nested)
	yubikey := devices[3]
	assert.Equal(t, "YubiKey OTP+FIDO+CCID", yubikey.Name)
	assert.Equal(t, "5.43", yubikey.Version)
	assert.Equal(t, "Yubico", yubikey.Vendor)

	// Thunderbolt
	tb := devices[4]
	assert.Equal(t, "Thunderbolt Bus", tb.Name)
	assert.Equal(t, "65.4.0", tb.Version)

	// Bluetooth
	bt := devices[5]
	assert.Equal(t, "Apple Bluetooth Module", bt.Name)
	assert.Equal(t, "22.1670.2.2", bt.Version)

	// Display/GPU
	gpu := devices[6]
	assert.Equal(t, "Apple M2 Max", gpu.Name)
	assert.Equal(t, "0x0001", gpu.Version)
}

func TestParseWindowsBIOS(t *testing.T) {
	f, err := os.Open("testdata/win32_bios.json")
	require.NoError(t, err)
	defer f.Close()

	devices, err := ParseWindowsBIOS(f)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	bios := devices[0]
	assert.Equal(t, "System BIOS", bios.Name)
	assert.Equal(t, "1.21.0", bios.Version)
	assert.Equal(t, "Dell Inc.", bios.Vendor)
	assert.Equal(t, "ABC1234", bios.DeviceId)
	assert.Equal(t, "Win32_BIOS", bios.Plugin)
}

func TestParseWindowsDiskDrive(t *testing.T) {
	f, err := os.Open("testdata/win32_diskdrive.json")
	require.NoError(t, err)
	defer f.Close()

	devices, err := ParseWindowsDiskDrive(f)
	require.NoError(t, err)
	require.Len(t, devices, 2)

	assert.Equal(t, "Samsung SSD 970 EVO Plus 1TB", devices[0].Name)
	assert.Equal(t, "2B2QEXM7", devices[0].Version)
	assert.Equal(t, "Samsung", devices[0].Vendor)

	assert.Equal(t, "WDC WD10EZEX-00WN4A0", devices[1].Name)
	assert.Equal(t, "01.01A01", devices[1].Version)
}

func TestParseWindowsVideoController(t *testing.T) {
	f, err := os.Open("testdata/win32_videocontroller.json")
	require.NoError(t, err)
	defer f.Close()

	devices, err := ParseWindowsVideoController(f)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	gpu := devices[0]
	assert.Equal(t, "NVIDIA GeForce RTX 4090", gpu.Name)
	assert.Equal(t, "31.0.15.5050", gpu.Version)
	assert.Equal(t, "NVIDIA", gpu.Vendor)
}

func TestGeneratePurl(t *testing.T) {
	tests := []struct {
		vendor  string
		name    string
		version string
		want    string
	}{
		{"Intel Corporation", "Management Engine", "14.1.53.1649", "pkg:generic/intel-corporation/management-engine@14.1.53.1649"},
		{"", "Boot ROM", "10151.140.19", "pkg:generic/boot-rom@10151.140.19"},
		{"Apple", "", "1.0", ""},
		{"", "", "", ""},
	}
	for _, tt := range tests {
		got := GeneratePurl(tt.vendor, tt.name, tt.version)
		assert.Equal(t, tt.want, got, "GeneratePurl(%q, %q, %q)", tt.vendor, tt.name, tt.version)
	}
}

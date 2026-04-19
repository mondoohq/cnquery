// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSMBIOSStrings(t *testing.T) {
	// "Dell Inc.\x00 1.21.0\x00 04/19/2024\x00\x00"
	data := []byte("Dell Inc.\x001.21.0\x0004/19/2024\x00\x00")
	strs := extractSMBIOSStrings(data)
	require.Len(t, strs, 3)
	assert.Equal(t, "Dell Inc.", strs[0])
	assert.Equal(t, "1.21.0", strs[1])
	assert.Equal(t, "04/19/2024", strs[2])
}

func TestSmbiosString(t *testing.T) {
	strs := []string{"Dell Inc.", "1.21.0", "04/19/2024"}
	assert.Equal(t, "Dell Inc.", smbiosString(strs, 1))
	assert.Equal(t, "1.21.0", smbiosString(strs, 2))
	assert.Equal(t, "04/19/2024", smbiosString(strs, 3))
	assert.Equal(t, "", smbiosString(strs, 0)) // 0 means "not set"
	assert.Equal(t, "", smbiosString(strs, 4)) // out of bounds
}

func TestParseSMBIOSBiosTable(t *testing.T) {
	// Build a minimal SMBIOS raw table:
	// 8-byte RawSMBIOSData header + Type 0 BIOS structure
	raw := make([]byte, 8) // header (all zeros is fine for testing)

	// Type 0 (BIOS Information) structure - 0x12 bytes formatted area
	bios := make([]byte, 0x12)
	bios[0] = 0x00 // Type: BIOS
	bios[1] = 0x12 // Length: 18 bytes
	bios[2] = 0x00 // Handle (low)
	bios[3] = 0x00 // Handle (high)
	bios[4] = 0x01 // Vendor string index = 1
	bios[5] = 0x02 // Version string index = 2
	bios[8] = 0x03 // Release date string index = 3

	raw = append(raw, bios...)
	// String table: "Dell Inc.\x00 1.21.0\x00 04/19/2024\x00\x00"
	raw = append(raw, []byte("Dell Inc.\x001.21.0\x0004/19/2024\x00\x00")...)

	dev := parseSMBIOSBiosTable(raw)
	require.NotNil(t, dev)
	assert.Equal(t, "System BIOS", dev.Name)
	assert.Equal(t, "1.21.0", dev.Version)
	assert.Equal(t, "Dell Inc.", dev.Vendor)
	assert.Equal(t, "04/19/2024", dev.DeviceId) // release date used as device ID
	assert.Equal(t, "GetSystemFirmwareTable", dev.Plugin)
	assert.NotEmpty(t, dev.Purl)
}

func TestExtractRevisionFromHWID(t *testing.T) {
	tests := []struct {
		hwID string
		want string
	}{
		{"PCI\\VEN_8086&DEV_9BC4&REV_05", "05"},
		{"PCI\\VEN_8086&DEV_9BC4&SUBSYS_17AA22C2&REV_05", "05"},
		{"USB\\VID_1050&PID_0407&REV_0543", "0543"},
		{"ACPI\\PNP0303", ""},
		{"PCI\\VEN_8086&DEV_9BC4", ""},
	}
	for _, tt := range tests {
		got := extractRevisionFromHWID(tt.hwID)
		assert.Equal(t, tt.want, got, "extractRevisionFromHWID(%q)", tt.hwID)
	}
}

func TestClassFromHWID(t *testing.T) {
	assert.Equal(t, "PCI", classFromHWID("PCI\\VEN_8086&DEV_9BC4"))
	assert.Equal(t, "USB", classFromHWID("USB\\VID_1050"))
	assert.Equal(t, "SCSI", classFromHWID("SCSI\\DISK"))
	assert.Equal(t, "noslash", classFromHWID("noslash"))
}

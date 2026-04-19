// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import "strings"

const (
	// SMBIOS table types
	smbiosTypeBIOS = 0
)

// parseSMBIOSBiosTable parses the raw SMBIOS firmware table and extracts
// Type 0 (BIOS Information) data. The input is the raw buffer returned by
// GetSystemFirmwareTable (Windows) with an 8-byte RawSMBIOSData header.
func parseSMBIOSBiosTable(raw []byte) *Device {
	if len(raw) < 8 {
		return nil
	}

	// Skip the 8-byte RawSMBIOSData header
	data := raw[8:]

	// Walk SMBIOS structures looking for Type 0 (BIOS)
	offset := 0
	for offset < len(data)-4 {
		structType := data[offset]
		structLen := int(data[offset+1])

		if structLen < 4 || offset+structLen > len(data) {
			break
		}

		if structType == smbiosTypeBIOS && structLen >= 0x12 {
			// Type 0: BIOS Information
			// Byte 4: Vendor (string index)
			// Byte 5: BIOS Version (string index)
			// Byte 8: BIOS Release Date (string index)
			vendorIdx := data[offset+4]
			versionIdx := data[offset+5]
			releaseDateIdx := data[offset+8]

			// Extract strings from the string table after the formatted area
			strs := extractSMBIOSStrings(data[offset+structLen:])

			vendor := smbiosString(strs, vendorIdx)
			version := smbiosString(strs, versionIdx)
			releaseDate := smbiosString(strs, releaseDateIdx)

			if version == "" {
				return nil
			}

			name := "System BIOS"
			return &Device{
				Name:     name,
				Version:  version,
				Vendor:   vendor,
				DeviceId: releaseDate,
				Summary:  "System BIOS/UEFI firmware (native)",
				Plugin:   "GetSystemFirmwareTable",
				Purl:     GeneratePurl(vendor, name, version),
			}
		}

		// Skip to next structure: walk past formatted area + string table
		// String table ends with double null (0x00, 0x00)
		strStart := offset + structLen
		for strStart < len(data)-1 {
			if data[strStart] == 0 && data[strStart+1] == 0 {
				strStart += 2
				break
			}
			strStart++
		}
		offset = strStart
	}

	return nil
}

// extractSMBIOSStrings reads null-terminated strings from the SMBIOS string
// table (which follows a structure's formatted area).
func extractSMBIOSStrings(data []byte) []string {
	var result []string
	start := 0
	for i, b := range data {
		if b == 0 {
			if i == start {
				// Double null = end of string table
				break
			}
			result = append(result, string(data[start:i]))
			start = i + 1
		}
	}
	return result
}

// smbiosString returns the 1-indexed string from the string table.
func smbiosString(strs []string, idx byte) string {
	if idx == 0 || int(idx) > len(strs) {
		return ""
	}
	return strs[idx-1]
}

// extractRevisionFromHWID extracts the REV_ value from a hardware ID string.
// Example: "PCI\VEN_8086&DEV_9BC4&REV_05" → "05"
func extractRevisionFromHWID(hwID string) string {
	upper := strings.ToUpper(hwID)
	for _, prefix := range []string{"REV_", "FW_"} {
		idx := strings.Index(upper, prefix)
		if idx < 0 {
			continue
		}
		start := idx + len(prefix)
		end := start
		for end < len(hwID) && hwID[end] != '&' && hwID[end] != '\\' {
			end++
		}
		if end > start {
			return hwID[start:end]
		}
	}
	return ""
}

// classFromHWID extracts the device class prefix from a hardware ID.
// Example: "PCI\VEN_8086&..." → "PCI"
func classFromHWID(hwID string) string {
	idx := strings.IndexByte(hwID, '\\')
	if idx > 0 {
		return hwID[:idx]
	}
	return hwID
}

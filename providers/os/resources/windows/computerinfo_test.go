// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsComputerInfo(t *testing.T) {
	r, err := os.Open("./testdata/computer-info.json")
	require.NoError(t, err)

	items, err := ParseComputerInfo(r)
	assert.Nil(t, err)
	assert.Equal(t, 43, len(items))
}

// TestParseCustomComputerInfoMapsTheRightSources pins the four fields that used
// to be wired to the wrong source. Each expected value is what Get-ComputerInfo
// reports for the same host, so the fallback and the native path agree.
func TestParseCustomComputerInfoMapsTheRightSources(t *testing.T) {
	raw := `{
      "Bios": {"SMBIOSBIOSVersion": "1.0", "Manufacturer": "Amazon EC2"},
      "ComputerSystem": {"TotalPhysicalMemory": 8482484224, "Model": "t3.large"},
      "Os": {"Version": "10.0.20348", "LastBootUpTime": "/Date(1786492800000)/", "BuildNumber": "20348"},
      "TimeZone": {"StandardName": "UTC"},
      "WindowsProduct": {"EditionID": "ServerDatacenter"},
      "FirmwareType": "Uefi",
      "Hal": "10.0.20348.5499",
      "PhysicalMemoryKB": 8388608,
      "Uptime": {"Ticks": 26530373913, "TotalSeconds": 2653.0373913}
    }`

	info, err := ParseCustomComputerInfo(strings.NewReader(raw))
	require.NoError(t, err)

	// Was read out of the WindowsProduct hive, which has no such value, so it
	// was always null even though the script had fetched it.
	assert.Equal(t, "Uefi", info["BiosFirmwareType"])

	// Was the operating system version. The HAL revision is the point of the
	// field and the OS version does not carry it.
	assert.Equal(t, "10.0.20348.5499", info["OsHardwareAbstractionLayer"])
	assert.NotEqual(t, info["OsVersion"], info["OsHardwareAbstractionLayer"])

	// Was TotalPhysicalMemory: a different quantity in a different unit, so
	// the number was wrong by roughly a factor of a thousand.
	assert.EqualValues(t, 8388608, info["CsPhyicallyInstalledMemory"])
	assert.NotEqual(t, info["CsTotalPhysicalMemory"], info["CsPhyicallyInstalledMemory"])

	// Was the boot timestamp, so a field documented as a duration carried a
	// point in time.
	uptime, ok := info["OsUptime"].(map[string]any)
	require.True(t, ok, "uptime must be a duration, not a timestamp string")
	assert.Contains(t, uptime, "TotalSeconds")

	// The fields that were already right must stay right.
	assert.Equal(t, "1.0", info["BiosSMBIOSBIOSVersion"])
	assert.Equal(t, "t3.large", info["CsModel"])
	assert.Equal(t, "ServerDatacenter", info["WindowsEditionId"])
	assert.EqualValues(t, 8482484224, info["CsTotalPhysicalMemory"])
}

// TestParseCustomComputerInfoAbsentSources keeps an unreadable source null
// rather than letting it fall back to a value from somewhere else.
func TestParseCustomComputerInfoAbsentSources(t *testing.T) {
	info, err := ParseCustomComputerInfo(strings.NewReader(
		`{"Bios": {}, "ComputerSystem": {}, "Os": {"Version": "10.0.14393"}, "TimeZone": {}, "WindowsProduct": {}}`))
	require.NoError(t, err)

	assert.Nil(t, info["BiosFirmwareType"])
	assert.Nil(t, info["OsHardwareAbstractionLayer"])
	assert.Nil(t, info["CsPhyicallyInstalledMemory"])
	assert.Nil(t, info["OsUptime"])
}

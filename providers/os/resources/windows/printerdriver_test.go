// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixture mirrors what Get-PrinterDriver | ConvertTo-Json emits: a packed
// 64-bit DriverVersion, the driver MODEL version in MajorVersion, and the INF
// the package was installed from.
const printerDriverJSON = `[
{"Name":"Microsoft Print To PDF","PrinterEnvironment":"Windows x64","Manufacturer":"Microsoft","DriverVersion":1688862492857958400,"MajorVersion":4,"InfPath":"C:\\Windows\\System32\\DriverStore\\FileRepository\\prnms009.inf_amd64_9d4f8b8bd0e4e0d5\\prnms009.inf","ConfigFile":"","DataFile":"C:\\Windows\\system32\\spool\\drivers\\x64\\3\\PrintToPDF.dll","DriverPath":"C:\\Windows\\system32\\spool\\drivers\\x64\\3\\mxdwdrv.dll","PrintProcessor":"winprint"},
{"Name":"Brother HL-L2375DW series","PrinterEnvironment":"Windows x64","Manufacturer":"Brother","DriverVersion":844433520133734400,"MajorVersion":3,"InfPath":"C:\\Windows\\System32\\DriverStore\\FileRepository\\brpom19a.inf_amd64_1a2b3c\\brpom19a.inf","ConfigFile":"BRPOM19A.DLL","DataFile":"BROM19A.DAT","DriverPath":"BRPRM19A.DLL","PrintProcessor":"winprint"}
]`

func TestParsePrinterDrivers(t *testing.T) {
	drivers, err := ParsePrinterDrivers(strings.NewReader(printerDriverJSON))
	require.NoError(t, err)
	require.Len(t, drivers, 2)

	assert.Equal(t, "Microsoft Print To PDF", drivers[0].Name)
	assert.Equal(t, "Windows x64", drivers[0].PrinterEnvironment)
	assert.Equal(t, int64(4), drivers[0].MajorVersion, "the driver MODEL version, not the vendor's release")
	assert.Contains(t, drivers[0].InfPath, "DriverStore")

	assert.Equal(t, "Brother", drivers[1].Manufacturer)
	assert.Equal(t, uint64(844433520133734400), drivers[1].DriverVersion,
		"the packed version must survive as an integer, not lose precision through a float")
}

// TestDottedVersionUnpacksTheWindowsEncoding: Windows packs a driver version
// into one 64-bit integer as four 16-bit fields, most significant first, so the
// raw value is meaningless on its own. Vendors publish the dotted form.
func TestDottedVersionUnpacksTheWindowsEncoding(t *testing.T) {
	for _, tc := range []struct {
		packed uint64
		want   string
	}{
		// 6.0.24328.0 — 6<<48 | 0<<32 | 24328<<16 | 0
		{6<<48 | 0<<32 | 24328<<16 | 0, "6.0.24328.0"},
		// 3.0.0.0, the shape a printer vendor advises on ("Ver.3.0.0.0")
		{3 << 48, "3.0.0.0"},
		// every field distinct, so a transposed shift is caught
		{1<<48 | 2<<32 | 3<<16 | 4, "1.2.3.4"},
		// the widest each field goes
		{0xFFFF<<48 | 0xFFFF<<32 | 0xFFFF<<16 | 0xFFFF, "65535.65535.65535.65535"},
	} {
		assert.Equal(t, tc.want, PrinterDriver{DriverVersion: tc.packed}.DottedVersion())
	}

	// Absent rather than "0.0.0.0": a driver with no version reported must not
	// look like a real version zero.
	assert.Empty(t, PrinterDriver{}.DottedVersion())
}

// TestParsePrinterDriversHandlesAnEmptyMachine: the spooler stopped, or no
// drivers installed, is a normal state and not a failure.
func TestParsePrinterDriversHandlesAnEmptyMachine(t *testing.T) {
	for _, in := range []string{"", "   ", "null"} {
		drivers, err := ParsePrinterDrivers(strings.NewReader(in))
		require.NoError(t, err, "input %q", in)
		assert.Empty(t, drivers)
	}
}

// TestParsePrinterDriversAcceptsAStringVersion: PowerShell sometimes serializes
// a large integer as a string. Both must decode to the same value.
func TestParsePrinterDriversAcceptsAStringVersion(t *testing.T) {
	drivers, err := ParsePrinterDrivers(strings.NewReader(
		`[{"Name":"x","DriverVersion":"844433520133734400","MajorVersion":"3"}]`))
	require.NoError(t, err)
	require.Len(t, drivers, 1)
	assert.Equal(t, uint64(844433520133734400), drivers[0].DriverVersion)
	assert.Equal(t, int64(3), drivers[0].MajorVersion)
}

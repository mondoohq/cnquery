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
	// Build the packed value from its four fields rather than writing the
	// shifts inline: an inline "a<<48 | 0<<32 | c<<16 | 0" reads as documenting
	// all four positions but the zero terms are no-ops, which staticcheck
	// rightly flags as a mistake waiting to happen.
	packed := func(a, b, c, d uint64) uint64 {
		return a<<48 | b<<32 | c<<16 | d
	}

	for _, tc := range []struct {
		packed uint64
		want   string
	}{
		{packed(6, 0, 24328, 0), "6.0.24328.0"},
		// The shape a printer vendor advises on ("Ver.3.0.0.0").
		{packed(3, 0, 0, 0), "3.0.0.0"},
		// Every field distinct, so a transposed shift is caught.
		{packed(1, 2, 3, 4), "1.2.3.4"},
		// The widest each field goes.
		{packed(0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF), "65535.65535.65535.65535"},
		// The high bit set: the value exceeds what a signed 64-bit integer can
		// hold, which is why this is exposed as a dotted string rather than as
		// a packed number.
		{packed(0x8000, 0, 0, 1), "32768.0.0.1"},
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

// TestPurlKeepsVendorsApart is the reason the vendor is the namespace.
//
// Driver names are NOT unique across vendors: "PCL 6 Driver" and
// "PostScript3 Driver" name page description languages that every printer
// vendor ships a driver for. Keying on the name alone would match one vendor's
// advisory against another vendor's driver.
func TestPurlKeepsVendorsApart(t *testing.T) {
	ricoh := PrinterDriver{Name: "PCL 6 Driver", Manufacturer: "RICOH", DriverVersion: 3 << 48}
	brother := PrinterDriver{Name: "PCL 6 Driver", Manufacturer: "Brother", DriverVersion: 3 << 48}

	assert.Equal(t, "pkg:windows-driver/ricoh/pcl-6-driver@3.0.0.0", ricoh.Purl())
	assert.Equal(t, "pkg:windows-driver/brother/pcl-6-driver@3.0.0.0", brother.Purl())
	assert.NotEqual(t, ricoh.Purl(), brother.Purl(), "same driver name, different vendors")
}

// TestVendorTokenNormalisesCorporateSuffixes: the spooler reports the
// manufacturer as the INF spells it, which carries corporate suffixes the
// vendor's advisories do not.
func TestVendorTokenNormalisesCorporateSuffixes(t *testing.T) {
	for _, in := range []string{"RICOH", "Ricoh", "Ricoh Company, Ltd.", "RICOH COMPANY, LTD."} {
		assert.Equal(t, "ricoh", VendorToken(in), "input %q", in)
	}
	for _, in := range []string{"Brother", "Brother Industries, Ltd", "brother industries"} {
		assert.Equal(t, "brother", VendorToken(in), "input %q", in)
	}
	assert.Equal(t, "konica-minolta", VendorToken("Konica Minolta, Inc."))
}

// TestPurlRefusesAPartialIdentity: a vendor-less driver PURL would match
// whatever advisory carried the same driver name, so an unidentifiable driver
// yields nothing at all.
func TestPurlRefusesAPartialIdentity(t *testing.T) {
	assert.Empty(t, PrinterDriver{Name: "PCL 6 Driver"}.Purl(), "no manufacturer")
	assert.Empty(t, PrinterDriver{Manufacturer: "RICOH"}.Purl(), "no name")
	assert.Empty(t, PrinterDriver{Name: "  ", Manufacturer: "  "}.Purl())

	// A driver with no version is still identifiable; the PURL simply carries
	// no version, and an unversioned package cannot match a bounded advisory.
	assert.Equal(t, "pkg:windows-driver/ricoh/lan-fax-driver",
		PrinterDriver{Name: "LAN Fax Driver", Manufacturer: "RICOH"}.Purl())
}

// TestPurlOnRealVendorDriverNames uses the names Ricoh publishes in its own
// advisories, which are what the catalog rows have to line up with.
func TestPurlOnRealVendorDriverNames(t *testing.T) {
	for name, want := range map[string]string{
		"PostScript3 Driver":                 "pkg:windows-driver/ricoh/postscript3-driver",
		"PCL 6 Driver":                       "pkg:windows-driver/ricoh/pcl-6-driver",
		"PS Driver for Universal Print":      "pkg:windows-driver/ricoh/ps-driver-for-universal-print",
		"PCL6 Driver for Universal Print":    "pkg:windows-driver/ricoh/pcl6-driver-for-universal-print",
		"PCL6 V4 Driver for Universal Print": "pkg:windows-driver/ricoh/pcl6-v4-driver-for-universal-print",
		"LAN Fax Driver":                     "pkg:windows-driver/ricoh/lan-fax-driver",
		"Generic PCL5 Driver":                "pkg:windows-driver/ricoh/generic-pcl5-driver",
	} {
		got := PrinterDriver{Name: name, Manufacturer: "RICOH"}.Purl()
		assert.Equal(t, want, got, "name %q", name)
	}
}

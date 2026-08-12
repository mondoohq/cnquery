// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PrinterDriversScript enumerates the print drivers installed on the machine.
//
// Get-PrinterDriver reports drivers registered with the spooler, which is what
// actually runs them — as opposed to the Add/Remove Programs list, where only
// vendor bundles that shipped an installer appear. A driver delivered as an
// INF/CAB package lands in the driver store and has no uninstall entry at all,
// so enumerating software would miss it entirely.
//
// -ErrorAction SilentlyContinue: the cmdlet is unavailable on Server Core
// installations without the printing feature, and a machine with no drivers is
// a normal state rather than a failure.
//
// ConvertTo-Json is forced to a list with @(): PowerShell serializes a single
// object as an object rather than a one-element array, which would otherwise
// need two parse paths.
const PrinterDriversScript = `
@(Get-PrinterDriver -ErrorAction SilentlyContinue | Select-Object -Property Name,PrinterEnvironment,Manufacturer,DriverVersion,MajorVersion,InfPath,ConfigFile,DataFile,DriverPath,PrintProcessor) | ConvertTo-Json -Compress
`

// PrinterDriver is one driver as the spooler reports it.
type PrinterDriver struct {
	Name string `json:"Name"`
	// PrinterEnvironment is the architecture the driver is registered under,
	// e.g. "Windows x64".
	PrinterEnvironment string `json:"PrinterEnvironment"`
	Manufacturer       string `json:"Manufacturer"`
	// DriverVersion is Windows' packed 64-bit version. See DottedVersion.
	DriverVersion uint64 `json:"DriverVersion"`
	// MajorVersion is the driver MODEL version (2, 3, 4) — the spooler
	// interface generation, not the vendor's release number.
	MajorVersion   int64  `json:"MajorVersion"`
	InfPath        string `json:"InfPath"`
	ConfigFile     string `json:"ConfigFile"`
	DataFile       string `json:"DataFile"`
	DriverPath     string `json:"DriverPath"`
	PrintProcessor string `json:"PrintProcessor"`
}

// DottedVersion renders DriverVersion the way a vendor writes it.
//
// Windows packs a driver version into one 64-bit integer as four 16-bit fields,
// most significant first, so the raw value is unreadable on its own:
//
//	1688862492857958400  ->  6.0.24328.0
//
// Vendors publish and advise on the dotted form ("Ver.3.0.0.0"), so a caller
// comparing against vendor documentation needs this rather than the packed
// integer. Both are exposed, because the packed value is what the spooler
// actually stores and is the one to use when comparing two drivers exactly.
func (d PrinterDriver) DottedVersion() string {
	if d.DriverVersion == 0 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		(d.DriverVersion>>48)&0xFFFF,
		(d.DriverVersion>>32)&0xFFFF,
		(d.DriverVersion>>16)&0xFFFF,
		d.DriverVersion&0xFFFF,
	)
}

// ParsePrinterDrivers reads the script's output.
//
// An empty document is not an error: a machine with the spooler stopped, or
// with no drivers installed, legitimately reports nothing.
func ParsePrinterDrivers(r io.Reader) ([]PrinterDriver, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return []PrinterDriver{}, nil
	}

	// DriverVersion arrives as a JSON number large enough to lose precision in
	// a float, and PowerShell occasionally emits it as a string. Decode into
	// json.Number and convert explicitly rather than letting either happen
	// silently.
	var raw []struct {
		PrinterDriver
		DriverVersion json.Number `json:"DriverVersion"`
		MajorVersion  json.Number `json:"MajorVersion"`
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, errors.New("failed to parse printer drivers: " + err.Error())
	}

	out := make([]PrinterDriver, 0, len(raw))
	for _, r := range raw {
		d := r.PrinterDriver
		if v, err := strconv.ParseUint(r.DriverVersion.String(), 10, 64); err == nil {
			d.DriverVersion = v
		}
		if v, err := strconv.ParseInt(r.MajorVersion.String(), 10, 64); err == nil {
			d.MajorVersion = v
		}
		out = append(out, d)
	}
	return out, nil
}

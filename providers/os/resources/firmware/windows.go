// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import (
	"encoding/json"
	"io"
)

// ParseWindowsBIOS parses the JSON output of:
// Get-CimInstance Win32_BIOS | ConvertTo-Json
func ParseWindowsBIOS(r io.Reader) ([]Device, error) {
	// PowerShell may return a single object or an array
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var items []winBIOS
	if err := json.Unmarshal(raw, &items); err != nil {
		// Try single object
		var single winBIOS
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, err
		}
		items = []winBIOS{single}
	}

	var devices []Device
	for _, b := range items {
		if b.Version == "" {
			continue
		}
		name := "System BIOS"
		devices = append(devices, Device{
			Name:     name,
			DeviceId: b.SerialNumber,
			Version:  b.Version,
			Vendor:   b.Manufacturer,
			Summary:  "System BIOS/UEFI firmware",
			Plugin:   "Win32_BIOS",
			Purl:     GeneratePurl(b.Manufacturer, name, b.Version),
		})
	}
	return devices, nil
}

type winBIOS struct {
	Version      string `json:"SMBIOSBIOSVersion"`
	Manufacturer string `json:"Manufacturer"`
	SerialNumber string `json:"SerialNumber"`
}

// ParseWindowsDiskDrive parses the JSON output of:
// Get-CimInstance Win32_DiskDrive | Select-Object Model, FirmwareRevision, SerialNumber, Manufacturer | ConvertTo-Json
func ParseWindowsDiskDrive(r io.Reader) ([]Device, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var items []winDiskDrive
	if err := json.Unmarshal(raw, &items); err != nil {
		var single winDiskDrive
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, err
		}
		items = []winDiskDrive{single}
	}

	var devices []Device
	for _, d := range items {
		if d.FirmwareRevision == "" {
			continue
		}
		name := d.Model
		if name == "" {
			name = "Disk Drive"
		}
		devices = append(devices, Device{
			Name:     name,
			DeviceId: d.SerialNumber,
			Version:  d.FirmwareRevision,
			Vendor:   d.Manufacturer,
			Summary:  "Storage device firmware",
			Plugin:   "Win32_DiskDrive",
			Purl:     GeneratePurl(d.Manufacturer, name, d.FirmwareRevision),
		})
	}
	return devices, nil
}

type winDiskDrive struct {
	Model            string `json:"Model"`
	FirmwareRevision string `json:"FirmwareRevision"`
	SerialNumber     string `json:"SerialNumber"`
	Manufacturer     string `json:"Manufacturer"`
}

// ParseWindowsVideoController parses the JSON output of:
// Get-CimInstance Win32_VideoController | Select-Object Name, DriverVersion, AdapterCompatibility | ConvertTo-Json
func ParseWindowsVideoController(r io.Reader) ([]Device, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var items []winVideoController
	if err := json.Unmarshal(raw, &items); err != nil {
		var single winVideoController
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, err
		}
		items = []winVideoController{single}
	}

	var devices []Device
	for _, v := range items {
		if v.DriverVersion == "" {
			continue
		}
		name := v.Name
		if name == "" {
			name = "Video Controller"
		}
		devices = append(devices, Device{
			Name:    name,
			Version: v.DriverVersion,
			Vendor:  v.AdapterCompatibility,
			Summary: "GPU driver/firmware",
			Plugin:  "Win32_VideoController",
			Purl:    GeneratePurl(v.AdapterCompatibility, name, v.DriverVersion),
		})
	}
	return devices, nil
}

type winVideoController struct {
	Name                 string `json:"Name"`
	DriverVersion        string `json:"DriverVersion"`
	AdapterCompatibility string `json:"AdapterCompatibility"`
}

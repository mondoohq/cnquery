// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import (
	"encoding/json"
	"io"
	"slices"
)

// fwupdOutput represents the top-level JSON output of `fwupdmgr get-devices --json`.
type fwupdOutput struct {
	Devices []fwupdDevice `json:"Devices"`
}

type fwupdDevice struct {
	Name          string   `json:"Name"`
	DeviceId      string   `json:"DeviceId"`
	Version       string   `json:"Version"`
	Vendor        string   `json:"Vendor"`
	VendorId      string   `json:"VendorId"`
	Summary       string   `json:"Summary"`
	Guid          []string `json:"Guid"`
	Plugin        string   `json:"Plugin"`
	Protocol      string   `json:"Protocol"`
	Flags         []string `json:"Flags"`
	VersionFormat string   `json:"VersionFormat"`
}

// ParseFwupd parses the JSON output of `fwupdmgr get-devices --json`.
func ParseFwupd(r io.Reader) ([]Device, error) {
	var out fwupdOutput
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, err
	}

	devices := make([]Device, 0, len(out.Devices))
	for _, d := range out.Devices {
		// Skip devices without a name (they are typically internal bus entries)
		if d.Name == "" {
			continue
		}

		dev := Device{
			Name:          d.Name,
			DeviceId:      d.DeviceId,
			Version:       d.Version,
			Vendor:        d.Vendor,
			VendorId:      d.VendorId,
			Summary:       d.Summary,
			Guid:          d.Guid,
			Plugin:        d.Plugin,
			Protocol:      d.Protocol,
			Flags:         d.Flags,
			VersionFormat: d.VersionFormat,
			Updatable:     slices.Contains(d.Flags, "updatable"),
			Purl:          GeneratePurl(d.Vendor, d.Name, d.Version),
		}
		devices = append(devices, dev)
	}

	return devices, nil
}

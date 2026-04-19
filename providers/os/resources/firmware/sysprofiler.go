// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package firmware

import (
	"encoding/json"
	"io"
)

// ParseSystemProfiler parses the JSON output of
// `system_profiler SPHardwareDataType SPNVMeDataType SPUSBDataType SPThunderboltDataType -json`.
// Each data type is a top-level key mapping to an array of items.
func ParseSystemProfiler(r io.Reader) ([]Device, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, err
	}

	var devices []Device

	if data, ok := raw["SPHardwareDataType"]; ok {
		devices = append(devices, parseHardware(data)...)
	}
	if data, ok := raw["SPNVMeDataType"]; ok {
		devices = append(devices, parseNVMe(data)...)
	}
	if data, ok := raw["SPUSBDataType"]; ok {
		devices = append(devices, parseUSB(data)...)
	}
	if data, ok := raw["SPThunderboltDataType"]; ok {
		devices = append(devices, parseThunderbolt(data)...)
	}
	if data, ok := raw["SPBluetoothDataType"]; ok {
		devices = append(devices, parseBluetooth(data)...)
	}
	if data, ok := raw["SPDisplaysDataType"]; ok {
		devices = append(devices, parseDisplays(data)...)
	}

	return devices, nil
}

// --- SPHardwareDataType ---

type spHardwareItem struct {
	Name           string `json:"_name"`
	BootRomVersion string `json:"boot_rom_version"`
	ModelID        string `json:"model_identifier"`
	ModelName      string `json:"machine_model"`
	SerialNumber   string `json:"serial_number"`
}

func parseHardware(data json.RawMessage) []Device {
	var items []spHardwareItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var devices []Device
	for _, item := range items {
		if item.BootRomVersion == "" {
			continue
		}
		name := "Boot ROM"
		if item.ModelName != "" {
			name = item.ModelName + " Boot ROM"
		}
		devices = append(devices, Device{
			Name:     name,
			DeviceId: item.ModelID,
			Version:  item.BootRomVersion,
			Vendor:   "Apple",
			Summary:  "System firmware (Boot ROM)",
			Plugin:   "SPHardwareDataType",
			Purl:     GeneratePurl("Apple", name, item.BootRomVersion),
		})
	}
	return devices
}

// --- SPNVMeDataType ---

type spNVMeItem struct {
	Name  string         `json:"_name"`
	Items []spNVMeDevice `json:"_items"`
	// Some versions put devices directly at the top level
	Revision string `json:"device_revision"`
	Model    string `json:"device_model"`
	Serial   string `json:"device_serial"`
}

type spNVMeDevice struct {
	Name     string `json:"_name"`
	Revision string `json:"device_revision"`
	Model    string `json:"device_model"`
	Serial   string `json:"device_serial"`
}

func parseNVMe(data json.RawMessage) []Device {
	var items []spNVMeItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var devices []Device
	for _, item := range items {
		// Check for nested items (controller → devices)
		for _, dev := range item.Items {
			if dev.Revision == "" {
				continue
			}
			name := dev.Name
			if name == "" {
				name = dev.Model
			}
			devices = append(devices, Device{
				Name:     name,
				DeviceId: dev.Serial,
				Version:  dev.Revision,
				Summary:  "NVMe storage controller",
				Plugin:   "SPNVMeDataType",
				Purl:     GeneratePurl("", name, dev.Revision),
			})
		}
		// Top-level device (when not nested)
		if item.Revision != "" && len(item.Items) == 0 {
			name := item.Name
			if name == "" {
				name = item.Model
			}
			devices = append(devices, Device{
				Name:     name,
				DeviceId: item.Serial,
				Version:  item.Revision,
				Summary:  "NVMe storage controller",
				Plugin:   "SPNVMeDataType",
				Purl:     GeneratePurl("", name, item.Revision),
			})
		}
	}
	return devices
}

// --- SPUSBDataType ---

type spUSBItem struct {
	Name         string      `json:"_name"`
	BcdDevice    string      `json:"bcd_device"`
	Manufacturer string      `json:"manufacturer"`
	VendorID     string      `json:"vendor_id"`
	ProductID    string      `json:"product_id"`
	Items        []spUSBItem `json:"_items"`
}

func parseUSB(data json.RawMessage) []Device {
	var items []spUSBItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var devices []Device
	collectUSB(&devices, items)
	return devices
}

func collectUSB(devices *[]Device, items []spUSBItem) {
	for _, item := range items {
		if item.BcdDevice != "" {
			*devices = append(*devices, Device{
				Name:     item.Name,
				Version:  item.BcdDevice,
				Vendor:   item.Manufacturer,
				VendorId: item.VendorID,
				Summary:  "USB device",
				Plugin:   "SPUSBDataType",
				Purl:     GeneratePurl(item.Manufacturer, item.Name, item.BcdDevice),
			})
		}
		// USB hubs can contain nested devices
		if len(item.Items) > 0 {
			collectUSB(devices, item.Items)
		}
	}
}

// --- SPThunderboltDataType ---

type spThunderboltItem struct {
	Name            string                `json:"_name"`
	FirmwareVersion string                `json:"device_firmware_version"`
	VendorName      string                `json:"vendor_name"`
	DeviceID        string                `json:"device_id_key"`
	Items           []spThunderboltDevice `json:"_items"`
}

type spThunderboltDevice struct {
	Name            string `json:"_name"`
	FirmwareVersion string `json:"device_firmware_version"`
	VendorName      string `json:"vendor_name"`
	DeviceID        string `json:"device_id_key"`
}

func parseThunderbolt(data json.RawMessage) []Device {
	var items []spThunderboltItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var devices []Device
	for _, item := range items {
		if item.FirmwareVersion != "" {
			devices = append(devices, Device{
				Name:     item.Name,
				DeviceId: item.DeviceID,
				Version:  item.FirmwareVersion,
				Vendor:   item.VendorName,
				Summary:  "Thunderbolt controller",
				Plugin:   "SPThunderboltDataType",
				Purl:     GeneratePurl(item.VendorName, item.Name, item.FirmwareVersion),
			})
		}
		for _, dev := range item.Items {
			if dev.FirmwareVersion == "" {
				continue
			}
			devices = append(devices, Device{
				Name:     dev.Name,
				DeviceId: dev.DeviceID,
				Version:  dev.FirmwareVersion,
				Vendor:   dev.VendorName,
				Summary:  "Thunderbolt device",
				Plugin:   "SPThunderboltDataType",
				Purl:     GeneratePurl(dev.VendorName, dev.Name, dev.FirmwareVersion),
			})
		}
	}
	return devices
}

// --- SPBluetoothDataType ---

type spBluetoothItem struct {
	Name            string            `json:"_name"`
	FirmwareVersion string            `json:"fw_version"`
	Controller      map[string]string `json:"controller_properties"`
}

func parseBluetooth(data json.RawMessage) []Device {
	var items []spBluetoothItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var devices []Device
	for _, item := range items {
		version := item.FirmwareVersion
		if version == "" && item.Controller != nil {
			version = item.Controller["fw_version"]
		}
		if version == "" {
			continue
		}
		name := item.Name
		if name == "" {
			name = "Bluetooth Controller"
		}
		devices = append(devices, Device{
			Name:    name,
			Version: version,
			Vendor:  "Apple",
			Summary: "Bluetooth controller",
			Plugin:  "SPBluetoothDataType",
			Purl:    GeneratePurl("Apple", name, version),
		})
	}
	return devices
}

// --- SPDisplaysDataType ---

type spDisplayItem struct {
	Name       string `json:"_name"`
	Vendor     string `json:"sppci_vendor"`
	RevisionID string `json:"spdisplays_revision-id"`
	DeviceID   string `json:"spdisplays_device-id"`
	RomVersion string `json:"spdisplays_rom_version"`
}

func parseDisplays(data json.RawMessage) []Device {
	var items []spDisplayItem
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	var devices []Device
	for _, item := range items {
		version := item.RomVersion
		if version == "" {
			version = item.RevisionID
		}
		if version == "" {
			continue
		}
		devices = append(devices, Device{
			Name:     item.Name,
			DeviceId: item.DeviceID,
			Version:  version,
			Vendor:   item.Vendor,
			Summary:  "GPU firmware",
			Plugin:   "SPDisplaysDataType",
			Purl:     GeneratePurl(item.Vendor, item.Name, version),
		})
	}
	return devices
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package usb

import (
	"fmt"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/resources/plist"
)

type USBDevice struct {
	Name             string
	VendorID         string
	ProductID        string
	Manufacturer     string
	Product          string
	SerialNumber     string
	DeviceClass      string
	DeviceClassName  string
	DeviceSubClass   string
	DeviceProtocol   string
	LocationID       string
	BusNumber        string
	DeviceAddress    string
	USBSpeed         string
	BcdDevice        string
	FormattedVersion string
	IsRemovable      bool
}

func ParseMacosIORegData(data any, devices *[]USBDevice) {
	// Process the data based on its type
	switch v := data.(type) {
	case plist.Data:
		// root object, we need to convert it to map[string]any
		obj := data.(plist.Data)
		// typecase so that we reach case map[string]any
		ParseMacosIORegData(map[string]any(obj), devices)
	case []any:
		// An array of entries
		for _, entry := range v {
			ParseMacosIORegData(entry, devices)
		}
	case map[string]any:
		// A single entry
		// Check if this is a USB device with the right properties
		if isUSBDevice(v) {
			device := extractDeviceInfo(v)
			*devices = append(*devices, device)
		}

		// Check if this entry has children
		if children, ok := v["IORegistryEntryChildren"]; ok {
			// Process children recursively
			ParseMacosIORegData(children, devices)
		}
	}
}

func isUSBDevice(entry map[string]any) bool {
	// Check for properties that indicate this is a USB device
	_, hasVendorID := entry["idVendor"]
	_, hasProductID := entry["idProduct"]

	// Also check the class name for USB indicators
	if className, ok := entry["IOClass"].(string); ok {
		return hasVendorID || hasProductID || strings.Contains(className, "USB")
	}

	return hasVendorID || hasProductID
}

func extractDeviceInfo(entry map[string]any) USBDevice {
	device := USBDevice{}

	// Extract name
	if name, ok := entry["IORegistryEntryName"].(string); ok {
		device.Name = name
	}

	// Extract USB identifiers
	if vendorID, ok := entry["idVendor"].(float64); ok {
		device.VendorID = fmt.Sprintf("0x%04x", int(vendorID))
	}

	if productID, ok := entry["idProduct"].(float64); ok {
		device.ProductID = fmt.Sprintf("0x%04x", int(productID))
	}

	// Extract device class info
	if deviceClass, ok := entry["bDeviceClass"].(float64); ok {
		device.DeviceClass = fmt.Sprintf("0x%02x", int(deviceClass))
		device.DeviceClassName = GetUSBClassDescription(device.DeviceClass)
	}

	if deviceSubClass, ok := entry["bDeviceSubClass"].(float64); ok {
		device.DeviceSubClass = fmt.Sprintf("0x%02x", int(deviceSubClass))
	}

	if deviceProtocol, ok := entry["bDeviceProtocol"].(float64); ok {
		device.DeviceProtocol = fmt.Sprintf("0x%02x", int(deviceProtocol))
	}

	// Extract descriptive strings
	if manufacturer, ok := entry["USB Vendor Name"].(string); ok {
		device.Manufacturer = manufacturer
	}

	if product, ok := entry["USB Product Name"].(string); ok {
		device.Product = product
	}

	if serial, ok := entry["USB Serial Number"].(string); ok {
		device.SerialNumber = serial
	}

	// Extract location info
	if locationID, ok := entry["locationID"].(float64); ok {
		device.LocationID = fmt.Sprintf("0x%08x", int(locationID))

		// Extract bus and address from location ID
		busNum := (int(locationID) >> 24) & 0xFF
		deviceAddr := int(locationID) & 0xFF

		device.BusNumber = fmt.Sprintf("%d", busNum)
		device.DeviceAddress = fmt.Sprintf("%d", deviceAddr)
	}

	// Extract speed
	if speed, ok := entry["USBSpeed"].(float64); ok {
		// Convert numeric speed to string with units
		device.USBSpeed = formatUSBSpeed(int(speed))
	}

	// Extract device version
	if bcdDevice, ok := entry["bcdDevice"].(float64); ok {
		device.BcdDevice = fmt.Sprintf("0x%04x", int(bcdDevice))
		device.FormattedVersion = formatBcdVersion(int(bcdDevice))
	}

	device.IsRemovable = isUSBDeviceRemovable(entry)
	return device
}

// GetUSBClassDescription returns a description for standard USB device class codes
// See https://www.usb.org/defined-class-codes
func GetUSBClassDescription(classCode string) string {
	// Remove 0x prefix if present
	classCode = strings.TrimPrefix(classCode, "0x")

	// Parse hex value
	classValue, err := strconv.ParseInt(classCode, 16, 64)
	if err != nil {
		return ""
	}

	switch classValue {
	case 0x00:
		return "Device"
	case 0x01:
		return "Audio"
	case 0x02:
		return "Communications and CDC Control"
	case 0x03:
		return "Human Interface Device (HID)"
	case 0x05:
		return "Physical"
	case 0x06:
		return "Image"
	case 0x07:
		return "Printer"
	case 0x08:
		return "Mass Storage"
	case 0x09:
		return "Hub"
	case 0x0A:
		return "CDC-Data"
	case 0x0B:
		return "Smart Card"
	case 0x0D:
		return "Content Security"
	case 0x0E:
		return "Video"
	case 0x0F:
		return "Personal Healthcare"
	case 0x10:
		return "Audio/Video Devices"
	case 0x11:
		return "Billboard Device Class"
	case 0x12:
		return "USB Type-C Bridge Class"
	case 0xDC:
		return "Diagnostic Device"
	case 0xE0:
		return "Wireless Controller"
	case 0xEF:
		return "Miscellaneous"
	case 0xFE:
		return "Application Specific"
	case 0xFF:
		return "Vendor Specific"
	default:
		return ""
	}
}

// Format USB speed based on the numeric value
func formatUSBSpeed(speed int) string {
	switch speed {
	case 1:
		return "1.5 Mbps (Low Speed)"
	case 12:
		return "12 Mbps (Full Speed)"
	case 480:
		return "480 Mbps (High Speed)"
	case 5000:
		return "5 Gbps (Super Speed)"
	case 10000:
		return "10 Gbps (Super Speed+)"
	case 20000:
		return "20 Gbps (Super Speed+ Gen 2x2)"
	default:
		return fmt.Sprintf("%d Mbps", speed)
	}
}

// Determine if a USB device is removable based on IOKit properties
func isUSBDeviceRemovable(entry map[string]any) bool {
	// Check the official IOKit "non-removable" property
	// If this property exists and is true, the device is explicitly marked as non-removable
	if nonRemovable, ok := entry["non-removable"].(bool); ok {
		return !nonRemovable
	}

	// If we can't determine definitively, default to true
	return true
}

// Format BCD (Binary Coded Decimal) version to human-readable format
func formatBcdVersion(bcdVersion int) string {
	// Extract major version (high byte)
	major := (bcdVersion >> 8) & 0xFF

	// Extract minor version (upper nibble of low byte)
	minor := (bcdVersion >> 4) & 0x0F

	// Extract subminor version (lower nibble of low byte)
	subminor := bcdVersion & 0x0F

	// Format as version string
	if subminor > 0 {
		return fmt.Sprintf("%d.%d.%d", major, minor, subminor)
	}
	return fmt.Sprintf("%d.%d", major, minor)
}

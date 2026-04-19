// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package firmware

import (
	"encoding/binary"
	"fmt"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemFirmwareTable = kernel32.NewProc("GetSystemFirmwareTable")

	setupapi                              = windows.NewLazySystemDLL("setupapi.dll")
	procSetupDiGetClassDevsW              = setupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInfo             = setupapi.NewProc("SetupDiEnumDeviceInfo")
	procSetupDiGetDeviceRegistryPropertyW = setupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	procSetupDiDestroyDeviceInfoList      = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
)

const (
	// SMBIOS table provider signature ('RSMB')
	smbiosSignature = 0x52534D42

	// SetupDi constants
	digcfPresent    = 0x00000002
	digcfAllClasses = 0x00000004

	// Device registry property keys
	spdrpDeviceDesc = 0x00000000
	spdrpHardwareID = 0x00000001
	spdrpMfg        = 0x0000000B
)

// spDevinfoData corresponds to SP_DEVINFO_DATA.
type spDevinfoData struct {
	cbSize    uint32
	classGuid windows.GUID
	devInst   uint32
	reserved  uintptr
}

// CollectNative gathers firmware information using native Win32 APIs.
// This avoids PowerShell overhead and works even when PowerShell execution
// policy is restricted.
func CollectNative() []Device {
	var devices []Device

	// Phase 1: SMBIOS BIOS table
	if bios := readSMBIOSBios(); bios != nil {
		devices = append(devices, *bios)
	}

	// Phase 2: PnP device enumeration for disk/GPU firmware
	pnp := enumeratePnPDevices()
	devices = append(devices, pnp...)

	return devices
}

// readSMBIOSBios reads the SMBIOS Type 0 (BIOS) table via GetSystemFirmwareTable.
func readSMBIOSBios() *Device {
	// First call: get required buffer size
	size, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(smbiosSignature),
		0,
		0,
		0,
	)
	if size == 0 {
		return nil
	}

	buf := make([]byte, size)
	ret, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(smbiosSignature),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		size,
	)
	if ret == 0 {
		return nil
	}

	return parseSMBIOSBiosTable(buf[:ret])
}

// enumeratePnPDevices uses SetupDi APIs to enumerate PnP devices and extract
// device description, manufacturer, and hardware IDs.
func enumeratePnPDevices() []Device {
	// Get all present devices
	hDevInfo, _, _ := procSetupDiGetClassDevsW.Call(
		0,
		0,
		0,
		uintptr(digcfPresent|digcfAllClasses),
	)
	if hDevInfo == uintptr(syscall.InvalidHandle) {
		return nil
	}
	defer procSetupDiDestroyDeviceInfoList.Call(hDevInfo)

	var devices []Device
	var devInfo spDevinfoData
	devInfo.cbSize = uint32(unsafe.Sizeof(devInfo))

	for i := uint32(0); ; i++ {
		ret, _, _ := procSetupDiEnumDeviceInfo.Call(
			hDevInfo,
			uintptr(i),
			uintptr(unsafe.Pointer(&devInfo)),
		)
		if ret == 0 {
			break
		}

		name := getDeviceProperty(hDevInfo, &devInfo, spdrpDeviceDesc)
		mfg := getDeviceProperty(hDevInfo, &devInfo, spdrpMfg)
		hwID := getDeviceProperty(hDevInfo, &devInfo, spdrpHardwareID)

		if name == "" || hwID == "" {
			continue
		}

		// Filter: only include devices with firmware-relevant class prefixes
		hwIDUpper := strings.ToUpper(hwID)
		isFirmwareRelevant := strings.HasPrefix(hwIDUpper, "PCI\\") ||
			strings.HasPrefix(hwIDUpper, "USB\\") ||
			strings.HasPrefix(hwIDUpper, "ACPI\\") ||
			strings.HasPrefix(hwIDUpper, "SCSI\\") ||
			strings.HasPrefix(hwIDUpper, "IDE\\") ||
			strings.HasPrefix(hwIDUpper, "STORAGE\\")

		if !isFirmwareRelevant {
			continue
		}

		// Only include if it looks like it has a firmware version in the HW ID
		// (e.g., contains REV_ or FW_)
		hasFirmwareHint := strings.Contains(hwIDUpper, "REV_") ||
			strings.Contains(hwIDUpper, "&REV_") ||
			strings.Contains(hwIDUpper, "FW_")

		if !hasFirmwareHint {
			continue
		}

		// Extract revision from hardware ID (e.g., PCI\VEN_8086&DEV_9BC4&REV_05 → "05")
		version := extractRevisionFromHWID(hwID)
		if version == "" {
			continue
		}

		dev := Device{
			Name:     name,
			Version:  version,
			Vendor:   mfg,
			VendorId: hwID,
			Summary:  fmt.Sprintf("PnP device (%s)", classFromHWID(hwID)),
			Plugin:   "SetupDi",
			Purl:     GeneratePurl(mfg, name, version),
		}
		devices = append(devices, dev)
	}

	return devices
}

// getDeviceProperty reads a string registry property for a device.
func getDeviceProperty(hDevInfo uintptr, devInfo *spDevinfoData, prop uint32) string {
	var dataType uint32
	var buf [1024]byte

	ret, _, _ := procSetupDiGetDeviceRegistryPropertyW.Call(
		hDevInfo,
		uintptr(unsafe.Pointer(devInfo)),
		uintptr(prop),
		uintptr(unsafe.Pointer(&dataType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if ret == 0 {
		return ""
	}

	// The property is returned as a UTF-16 string
	return utf16ToString(buf[:])
}

// utf16ToString converts a null-terminated UTF-16LE byte slice to a Go string.
func utf16ToString(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	// Find null terminator
	for i, v := range u16 {
		if v == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}

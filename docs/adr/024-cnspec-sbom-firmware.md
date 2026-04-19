# ADR: mql Firmware SBOM Collection

**Status:** Experimental
**Date:** 2026-04-16

---

## Context

Firmware vulnerabilities are a critical attack surface — they persist across OS reinstalls, execute below the OS security boundary, and are difficult to detect with traditional tools. NIST SP 800-193 recommends organizations maintain an inventory of firmware components for all devices. Currently, mql surfaces SMBIOS data (`machine.bios`, `machine.system`, `machine.baseboard`) which provides static hardware identity, but does not enumerate the firmware versions running on individual components like storage controllers, network adapters, GPUs, Thunderbolt controllers, TPMs, and embedded controllers.

## Decision

Add `firmware.devices` and `firmware.device` as new MQL resources in the OS provider. These expose the per-component firmware inventory across Linux, macOS, and Windows using platform-native data sources, enabling SBOM generation that includes firmware as first-class packages.

## Data Gathering

### Linux: `fwupdmgr get-devices --json`

The Linux Firmware Update Daemon (fwupd) maintains a runtime inventory of all devices with updatable firmware. It is installed by default on Ubuntu, Fedora, RHEL 9+, and most desktop Linux distributions. A single CLI call returns all devices with firmware metadata:

```json
{
  "Devices": [
    {
      "Name": "Intel Management Engine",
      "DeviceId": "5fed1486be004d67ea79838d2e83aaa11bb72645",
      "Vendor": "Intel Corporation",
      "VendorId": "MEI:0x8086",
      "Version": "14.1.53.1649",
      "VersionFormat": "intel-me",
      "Guid": ["2800f812-b7b4-2d4b-aca8-46e0ff65814c"],
      "Summary": "Hardware and firmware technology for remote out-of-band management",
      "Plugin": "mei",
      "Protocol": "org.uefi.capsule",
      "Flags": ["internal", "registered", "can-verify"]
    }
  ]
}
```

Key fields for SBOM:
- **Name + Version**: Identify the firmware component and its installed version
- **Vendor / VendorId**: Identify the hardware manufacturer
- **Guid**: Unique device identifiers used by fwupd for update matching
- **DeviceId**: Stable SHA-1 hash identifying the device instance
- **Flags**: Indicate device capabilities (`updatable`, `can-verify`, `needs-reboot`, etc.)

**Fallback:** When `fwupdmgr` is not available but fwupd is running, data could be read via D-Bus at `org.freedesktop.fwupd`. However, D-Bus access requires a running daemon, so there is no reliable filesystem fallback. The initial implementation focuses on CLI-only.

### macOS: `system_profiler -json`

macOS `system_profiler` provides firmware information across multiple data types. Each is queried separately and the results are merged into the unified `firmware.device` schema:

| Data Type | Coverage | Key Fields |
|-----------|----------|------------|
| `SPHardwareDataType` | Boot ROM, Model, Serial | `boot_rom_version`, `model_identifier` |
| `SPNVMeDataType` | NVMe storage controller firmware | `device_revision`, `device_model`, `device_serial` |
| `SPUSBDataType` | USB device firmware | `bcd_device`, `manufacturer`, `vendor_id`, `product_id` |
| `SPThunderboltDataType` | Thunderbolt controller firmware | `device_firmware_version`, `vendor_name` |
| `SPBluetoothDataType` | Bluetooth firmware | `fw_version` |
| `SPDisplaysDataType` | GPU firmware/ROM revision | `spdisplays_revision-id`, `sppci_vendor` |

A single call can batch multiple data types:
```bash
system_profiler SPHardwareDataType SPNVMeDataType SPUSBDataType SPThunderboltDataType -json
```

### Windows: WMI / CIM

Windows exposes firmware information through CIM/WMI classes, queried via PowerShell:

| CIM Class | Coverage | Key Fields |
|-----------|----------|------------|
| `Win32_BIOS` | BIOS/UEFI firmware | `SMBIOSBIOSVersion`, `Manufacturer`, `SerialNumber` |
| `Win32_PnPSignedDriver` | All PnP device drivers/firmware | `DriverVersion`, `DeviceName`, `Manufacturer`, `HardWareID` |
| `Win32_VideoController` | GPU firmware | `DriverVersion`, `Name`, `AdapterCompatibility` |
| `Win32_NetworkAdapter` | NIC firmware | `Name`, `Manufacturer`, `MACAddress` |
| `Win32_DiskDrive` | Storage firmware | `FirmwareRevision`, `Model`, `SerialNumber` |

Example query (remote):
```powershell
Get-CimInstance Win32_BIOS | Select-Object SMBIOSBIOSVersion, Manufacturer, SerialNumber | ConvertTo-Json
Get-CimInstance Win32_DiskDrive | Select-Object Model, FirmwareRevision, SerialNumber | ConvertTo-Json
```

### Windows: Native syscall (local)

When running locally on Windows, firmware data can be retrieved directly via Win32 API syscalls without spawning PowerShell, using:

- **`GetSystemFirmwareTable`** — reads raw SMBIOS tables (Type 0: BIOS, Type 2: Baseboard, Type 3: Chassis) with version, vendor, and serial number
- **`SetupDiGetDeviceRegistryProperty`** (setupapi.h) — enumerates PnP devices with hardware IDs, driver versions, and manufacturer info
- **`DeviceIoControl`** with `IOCTL_STORAGE_QUERY_PROPERTY` — retrieves storage device firmware revision directly from the driver

The native approach avoids PowerShell startup overhead and works in environments where PowerShell execution policy is restricted. The PowerShell/CIM variant is used as fallback for remote connections via WinRM.

### Availability Detection

The resource gracefully returns an empty list when the platform-specific tool is unavailable:
- **Linux**: `fwupdmgr` not installed or fwupd daemon not running
- **macOS**: `system_profiler` is always available (ships with macOS)
- **Windows**: WMI/CIM is always available (ships with Windows)

## Resource Schema

### `firmware.device` (12 fields)

| Field | Type | Source (Linux / macOS / Windows) |
|-------|------|----------------------------------|
| `name` | string | fwupd Name / system_profiler device name / CIM DeviceName |
| `deviceId` | string | fwupd DeviceId / IORegistry path / PnP DeviceID |
| `version` | string | fwupd Version / firmware revision / DriverVersion |
| `vendor` | string | fwupd Vendor / manufacturer / Manufacturer |
| `vendorId` | string | fwupd VendorId / vendor_id / HardwareID prefix |
| `summary` | string | fwupd Summary / data type description / device class |
| `guid` | []string | fwupd Guid / IORegistry UUID / PnP HardwareIDs |
| `plugin` | string | fwupd Plugin / data type name / CIM class name |
| `protocol` | string | fwupd Protocol / n/a / n/a |
| `flags` | []string | fwupd Flags / n/a / n/a |
| `versionFormat` | string | fwupd VersionFormat / n/a / n/a |
| `updatable` | bool | fwupd "updatable" flag / n/a / n/a |

Note: Some fields (protocol, flags, versionFormat, updatable) are Linux-specific since fwupd provides richer metadata than macOS/Windows sources. These fields will be empty strings / empty lists / false on other platforms.

### `firmware` (1 field)

| Field | Type | Source |
|-------|------|--------|
| `devices` | []firmware.device | All enumerated firmware devices |

## PURL Generation

Firmware packages use the `generic` PURL type since there is no firmware-specific PURL type in the specification:

```
pkg:generic/<vendor>/<name>@<version>
```

Example: `pkg:generic/intel/management-engine@14.1.53.1649`

The vendor and name are normalized to lowercase with spaces replaced by hyphens.

## SBOM Integration

Each `firmware.device` with a non-empty version is emitted as a `BomPackage` with:
- **Name**: Device name
- **Version**: Firmware version
- **Format**: `"firmware"`
- **PUrl**: Generated as above

## Transport Compatibility

| Transport | Linux | macOS | Windows |
|-----------|-------|-------|---------|
| Local | `fwupdmgr` | `system_profiler` | Native syscalls (Win32 API) |
| SSH / WinRM | `fwupdmgr` via SSH | `system_profiler` via SSH | PowerShell CIM via WinRM |
| Container image | Not applicable (firmware is host-level) | N/A | N/A |

## Implementation

All three platforms (Linux, macOS, Windows) are implemented in a single PR to ensure the unified `firmware.device` schema works consistently across operating systems.

## Verification

- Unit tests with fixture JSON per platform
- Interactive: `mql run os -c "firmware.devices { list { name version vendor } }"`
- SBOM: `cnspec sbom -o cyclonedx-json | jq '.components[] | select(.type == "firmware")'`

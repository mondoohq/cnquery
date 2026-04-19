// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/firmware"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlFirmware) id() (string, error) {
	return "firmware", nil
}

func (r *mqlFirmware) devices() ([]any, error) {
	conn := r.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform

	var devices []firmware.Device

	switch {
	case pf.IsFamily("linux"):
		devices = r.collectLinux(conn)
	case pf.IsFamily("darwin"):
		devices = r.collectMacOS(conn)
	case pf.IsFamily("windows"):
		devices = r.collectWindows(conn)
	default:
		log.Debug().Str("platform", pf.Name).Msg("mql[firmware]> unsupported platform")
		return []any{}, nil
	}

	return r.toMqlDevices(devices)
}

func (r *mqlFirmware) collectLinux(conn shared.Connection) []firmware.Device {
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		log.Debug().Msg("mql[firmware]> no run command capability, skipping fwupd")
		return nil
	}

	cmd, err := conn.RunCommand("fwupdmgr get-devices --json")
	if err != nil || cmd.ExitStatus != 0 {
		log.Debug().Err(err).Msg("mql[firmware]> fwupdmgr not available or failed")
		return nil
	}

	devices, err := firmware.ParseFwupd(cmd.Stdout)
	if err != nil {
		log.Debug().Err(err).Msg("mql[firmware]> failed to parse fwupd output")
		return nil
	}
	return devices
}

func (r *mqlFirmware) collectMacOS(conn shared.Connection) []firmware.Device {
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		log.Debug().Msg("mql[firmware]> no run command capability, skipping system_profiler")
		return nil
	}

	cmd, err := conn.RunCommand("system_profiler SPHardwareDataType SPNVMeDataType SPUSBDataType SPThunderboltDataType SPBluetoothDataType SPDisplaysDataType -json")
	if err != nil || cmd.ExitStatus != 0 {
		log.Debug().Err(err).Msg("mql[firmware]> system_profiler failed")
		return nil
	}

	devices, err := firmware.ParseSystemProfiler(cmd.Stdout)
	if err != nil {
		log.Debug().Err(err).Msg("mql[firmware]> failed to parse system_profiler output")
		return nil
	}
	return devices
}

func (r *mqlFirmware) collectWindows(conn shared.Connection) []firmware.Device {
	// Local connection: use native Win32 APIs (no PowerShell overhead)
	if conn.Type() == shared.Type_Local {
		devices := firmware.CollectNative()
		if len(devices) > 0 {
			return devices
		}
		log.Debug().Msg("mql[firmware]> native Windows API returned no devices, falling back to PowerShell")
	}

	// Remote (WinRM) or native fallback: use PowerShell CIM queries
	return r.collectWindowsPowerShell(conn)
}

func (r *mqlFirmware) collectWindowsPowerShell(conn shared.Connection) []firmware.Device {
	if !conn.Capabilities().Has(shared.Capability_RunCommand) {
		log.Debug().Msg("mql[firmware]> no run command capability, skipping Windows firmware")
		return nil
	}

	var devices []firmware.Device

	// BIOS
	cmd, err := conn.RunCommand("powershell -NoProfile -Command \"Get-CimInstance Win32_BIOS | ConvertTo-Json\"")
	if err == nil && cmd.ExitStatus == 0 {
		bios, err := firmware.ParseWindowsBIOS(cmd.Stdout)
		if err == nil {
			devices = append(devices, bios...)
		} else {
			log.Debug().Err(err).Msg("mql[firmware]> failed to parse Win32_BIOS")
		}
	}

	// Disk drives
	cmd, err = conn.RunCommand("powershell -NoProfile -Command \"Get-CimInstance Win32_DiskDrive | Select-Object Model, FirmwareRevision, SerialNumber, Manufacturer | ConvertTo-Json\"")
	if err == nil && cmd.ExitStatus == 0 {
		disks, err := firmware.ParseWindowsDiskDrive(cmd.Stdout)
		if err == nil {
			devices = append(devices, disks...)
		} else {
			log.Debug().Err(err).Msg("mql[firmware]> failed to parse Win32_DiskDrive")
		}
	}

	// Video controllers
	cmd, err = conn.RunCommand("powershell -NoProfile -Command \"Get-CimInstance Win32_VideoController | Select-Object Name, DriverVersion, AdapterCompatibility | ConvertTo-Json\"")
	if err == nil && cmd.ExitStatus == 0 {
		gpus, err := firmware.ParseWindowsVideoController(cmd.Stdout)
		if err == nil {
			devices = append(devices, gpus...)
		} else {
			log.Debug().Err(err).Msg("mql[firmware]> failed to parse Win32_VideoController")
		}
	}

	return devices
}

func (r *mqlFirmware) toMqlDevices(devices []firmware.Device) ([]any, error) {
	mqlDevices := make([]any, 0, len(devices))
	for _, dev := range devices {
		guid := make([]any, len(dev.Guid))
		for i, g := range dev.Guid {
			guid[i] = g
		}
		flags := make([]any, len(dev.Flags))
		for i, f := range dev.Flags {
			flags[i] = f
		}

		id := "firmware.device/" + dev.DeviceId
		if dev.DeviceId == "" {
			id = "firmware.device/" + dev.Vendor + "/" + dev.Name + "@" + dev.Version
		}

		mqlDev, err := CreateResource(r.MqlRuntime, "firmware.device", map[string]*llx.RawData{
			"__id":          llx.StringData(id),
			"name":          llx.StringData(dev.Name),
			"deviceId":      llx.StringData(dev.DeviceId),
			"version":       llx.StringData(dev.Version),
			"vendor":        llx.StringData(dev.Vendor),
			"vendorId":      llx.StringData(dev.VendorId),
			"summary":       llx.StringData(dev.Summary),
			"guid":          llx.ArrayData(guid, types.String),
			"plugin":        llx.StringData(dev.Plugin),
			"protocol":      llx.StringData(dev.Protocol),
			"flags":         llx.ArrayData(flags, types.String),
			"versionFormat": llx.StringData(dev.VersionFormat),
			"updatable":     llx.BoolData(dev.Updatable),
			"purl":          llx.StringData(dev.Purl),
		})
		if err != nil {
			return nil, err
		}
		mqlDevices = append(mqlDevices, mqlDev)
	}
	return mqlDevices, nil
}

func (r *mqlFirmwareDevice) id() (string, error) {
	id := "firmware.device/" + r.DeviceId.Data
	if r.DeviceId.Data == "" {
		id = "firmware.device/" + r.Vendor.Data + "/" + r.Name.Data + "@" + r.Version.Data
	}
	return id, nil
}

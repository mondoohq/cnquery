// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import "github.com/rs/zerolog/log"

// Model + manufacturer from Win32_ComputerSystem, SMBIOS version from Win32_BIOS.
const windowsDetectionCommand = `$cs = Get-CimInstance -ClassName Win32_ComputerSystem; ` +
	`$bios = Get-CimInstance -ClassName Win32_BIOS; ` +
	`"$($cs.Model)|$($cs.Manufacturer)|$($bios.SMBIOSBIOSVersion)"`

// detectWindowsHypervisor detects the hypervisor on Windows.
func (h *hyper) detectWindowsHypervisor() (string, bool) {
	stdout, err := h.RunCommand(windowsDetectionCommand)
	if err != nil {
		log.Debug().Err(err).Msg("could not detect hypervisor")
		return "", false
	}
	return mapHypervisor(stdout)
}

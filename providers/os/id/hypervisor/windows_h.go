// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import "github.com/rs/zerolog/log"

var windowsDetectionCommands = []string{
	// System model is the preferred detection
	"wmic computersystem get model",
	// Fallback to check the computer manufacturer
	"Get-CimInstance -ClassName Win32_ComputerSystem | Select-Object -ExpandProperty Manufacturer",
	// Modern configurations like Windows Server 2022 running Hyper-V return generic data
	// so we need a more reliable method, we are going to check the SMBIOSBIOSVersion as
	// our last detection
	"Get-CimInstance -ClassName Win32_BIOS | Select-Object -ExpandProperty SMBIOSBIOSVersion",
}

// detectWindowsHypervisor detects the hypervisor on Windows.
func (h *hyper) detectWindowsHypervisor() (string, bool) {
	for _, command := range windowsDetectionCommands {
		stdout, err := h.RunCommand(command)
		if err == nil {
			if hypervisor, ok := mapHypervisor(stdout); ok {
				return hypervisor, ok
			}
		}

		log.Debug().Err(err).Str("command", command).Msg("could not detect hypervisor")
	}

	return "", false
}

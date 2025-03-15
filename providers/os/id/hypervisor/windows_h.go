// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

// detectWindowsHypervisor detects the hypervisor on Windows.
func (h *hyper) detectWindowsHypervisor() (string, bool) {
	stdout, err := h.RunCommand("wmic computersystem get model")
	if err == nil {
		if hypervisor, ok := mapHypervisor(stdout); ok {
			return hypervisor, ok
		}
	}

	// Use PowerShell as fallback
	stdout, err = h.RunCommand(
		"Get-CimInstance -ClassName Win32_ComputerSystem | Select-Object -ExpandProperty Manufacturer",
	)
	if err == nil {
		return mapHypervisor(stdout)
	}

	return "", false
}

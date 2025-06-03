// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import (
	"strings"
)

// detectDarwinHypervisor detects the hypervisor on Darwin.
func (h *hyper) detectDarwinHypervisor() (hypervisor string, ok bool) {
	value, err := h.RunCommand("sysctl -n machdep.cpu.features")
	if err != nil {
		return
	}
	if strings.Contains(value, "VMM") {
		return h.detectDarwinIOReg()
	}

	// This setting can be only "0" or "1"
	value, err = h.RunCommand("sysctl -n kern.hv_vmm_present")
	if err != nil {
		return
	}
	if value == "1" {
		return h.detectDarwinIOReg()
	}

	// Look at the model identifier
	return h.detectDarwinModelIdentifier()
}

// detectDarwinIOReg uses ioreg to detect virtualization.
func (h *hyper) detectDarwinIOReg() (string, bool) {
	stdout, err := h.RunCommand("ioreg -lw0")
	if err != nil {
		return "", false
	}
	return mapHypervisor(stdout)
}

// detectDarwinModelIdentifier uses system_profiler to detect virtualization.
func (h *hyper) detectDarwinModelIdentifier() (string, bool) {
	stdout, err := h.RunCommand("system_profiler SPHardwareDataType")
	if err != nil {
		return "", false
	}
	return mapHypervisor(stdout)
}

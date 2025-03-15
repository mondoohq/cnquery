// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import (
	"strings"
)

// detectDarwinHypervisor detects the hypervisor on Darwin.
func (h *hyper) detectDarwinHypervisor() (hypervisor string, ok bool) {
	stdout, err := h.RunCommand("sysctl -n machdep.cpu.features")
	if err != nil {
		return
	}
	if strings.Contains(stdout, "VMM") {
		return h.detectDarwinIOReg()
	}

	return "", false
}

// detectDarwinIOReg uses ioreg to detect virtualization.
func (h *hyper) detectDarwinIOReg() (string, bool) {
	stdout, err := h.RunCommand("ioreg -lw0")
	if err != nil {
		return "", false
	}
	return mapHypervisor(stdout)
}

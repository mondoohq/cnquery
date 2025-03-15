// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import (
	"bytes"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// List of Desktop Management Interface (DMI) files that might contain information
// related to the presence of a virtualization platform.
var dmiFilesSlice = []string{
	"/sys/class/dmi/id/product_name",
	"/sys/class/dmi/id/sys_vendor",
	"/sys/class/dmi/id/board_vendor",
	"/sys/class/dmi/id/bios_vendor",
	"/sys/class/dmi/id/product_version",
}

// detectLinuxHypervisor detects the hypervisor on Linux.
func (h *hyper) detectLinuxHypervisor() (hypervisor string, ok bool) {
	detectors := []func() (string, bool){
		h.detectSystemdDetectVirt,
		h.detectDMIVendor,
		h.detectDMIDecode,
		h.detectLXC,
	}
	// check for CPU "hypervisor" flag
	if h.detectLinuxCPUHypervisor() {
		for _, detectFn := range detectors {
			hypervisor, ok = detectFn()
			if ok {
				break
			}
		}
	}
	return
}

// detectLinuxCPUHypervisor detects if the CPU has the "hypervisor" flag.
func (h *hyper) detectLinuxCPUHypervisor() bool {
	content, err := afero.ReadFile(h.connection.FileSystem(), "/proc/cpuinfo")
	if err != nil {
		return false
	}
	return bytes.Contains(content, []byte("hypervisor"))
}

// detectSystemdDetectVirt runs "systemd-detect-virt" to identify the hypervisor.
func (h *hyper) detectSystemdDetectVirt() (string, bool) {
	systemdVirt, err := h.RunCommand("systemd-detect-virt")
	if err != nil {
		return "", false
	}
	return mapHypervisor(systemdVirt)
}

// detectDMIVendor checks known VM vendors in DMI data.
//
// This approach was inspired on systemd's work for a similar purpose.
// https://github.com/systemd/systemd/blob/main/src/basic/virt.c#L163
func (h *hyper) detectDMIVendor() (string, bool) {
	for _, f := range dmiFilesSlice {
		content, err := afero.ReadFile(h.connection.FileSystem(), f)
		if err != nil {
			continue
		}
		if name, ok := mapHypervisor(string(content)); ok {
			log.Debug().Str("file", f).Msg("os.id.hypervisor> found in dmi data")
			return name, true
		}
	}

	return "", false
}

// detectDMIDecode runs "dmidecode" and extracts hypervisor information.
func (h *hyper) detectDMIDecode() (string, bool) {
	productName, err := h.RunCommand("dmidecode -s system-product-name")
	if err != nil {
		return "", false
	}
	return mapHypervisor(productName)
}

// detectLXC detects LXC/LXD containers.
func (h *hyper) detectLXC() (string, bool) {
	content, err := afero.ReadFile(h.connection.FileSystem(), "/proc/1/environ")
	if err == nil && bytes.Contains(content, []byte("container=lxc")) {
		return "LXC/LXD", true
	}
	return "", false
}

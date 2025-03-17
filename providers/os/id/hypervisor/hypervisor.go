// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import (
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

// list of known hypervisors
var knownHypervisors = map[string]string{
	"vmware":        "VMware",
	"virtualbox":    "VirtualBox",
	"kvm":           "KVM",
	"qemu":          "QEMU",
	"rhev":          "RHEV Hypervisor",
	"hyper-v":       "Hyper-V",
	"xen":           "Xen",
	"parallels":     "Parallels",
	"bhyve":         "bhyve",
	"proxmox":       "Proxmox VE",
	"openvz":        "OpenVZ",
	"virtuozzo":     "Virtuozzo",
	"powervm":       "IBM PowerVM",
	"applevirtual":  "Apple Virtualization",
	"apple virtual": "Apple Virtualization",
}

// hyper is a helper struct to avoid passing the connection and platform
// as function arguments.
type hyper struct {
	connection shared.Connection
	platform   *inventory.Platform
}

// Hypervisor returns the hypervisor of the system.
func Hypervisor(conn shared.Connection, pf *inventory.Platform) (hypervisor string, ok bool) {
	if !pf.IsFamily(inventory.FAMILY_UNIX) && !pf.IsFamily(inventory.FAMILY_WINDOWS) {
		log.Warn().Msg("your platform is not supported for hypervisor detection")
		return
	}

	hype := &hyper{conn, pf}

	if pf.IsFamily(inventory.FAMILY_LINUX) {
		return hype.detectLinuxHypervisor()
	}
	if pf.IsFamily(inventory.FAMILY_DARWIN) {
		return hype.detectDarwinHypervisor()
	}
	if pf.IsFamily(inventory.FAMILY_WINDOWS) && conn.Capabilities().Has(shared.Capability_File) {
		return hype.detectWindowsHypervisor()
	}

	return
}

// mapHypervisor maps known hypervisors to their names.
func mapHypervisor(info string) (string, bool) {
	// make sure it is lower case
	info = strings.ToLower(info)

	for key, value := range knownHypervisors {
		if strings.Contains(info, key) {
			return value, true
		}
	}
	return "", false
}

// runCommand is a wrapper around connection.RunCommand that helps execute commands
// and read the standard output for unix and windows systems.
func (h *hyper) RunCommand(commandString string) (string, error) {
	if h.platform.IsFamily(inventory.FAMILY_WINDOWS) {
		commandString = powershell.Encode(commandString)
	}
	cmd, err := h.connection.RunCommand(commandString)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

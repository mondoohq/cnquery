// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import (
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

// list of known hypervisors
var knownHypervisors = map[string]string{
	"vmware":                   "VMware",
	"virtualbox":               "VirtualBox",
	"kvm":                      "KVM",
	"qemu":                     "QEMU",
	"rhev":                     "RHEV Hypervisor",
	"hyper-v":                  "Hyper-V",
	"xen":                      "Xen",
	"parallels":                "Parallels",
	"bhyve":                    "bhyve",
	"proxmox":                  "Proxmox VE",
	"openvz":                   "OpenVZ",
	"virtuozzo":                "Virtuozzo",
	"powervm":                  "IBM PowerVM",
	"applevirtual":             "Apple Virtualization",
	"apple virtual":            "Apple Virtualization",
	"nutanix":                  "Nutanix Acropolis",
	"openshift virtualization": "OpenShift Virtualization",
	"red hat":                  "OpenShift Virtualization",

	// The keys below are the identifiers systemd-detect-virt actually emits,
	// which differ from the product names above. Without them a guest is
	// detected and then fails to map, so os.hypervisor reads null.
	// See src/basic/virt.c, virtualization_table.
	//
	// AWS Nitro. systemd reports "amazon" for Nitro and "xen" for the older
	// Xen instance families, and the DMI vendor fields are disjoint the same
	// way ("Amazon EC2" against "Xen"), so this cannot shadow xen.
	"amazon": "AWS Nitro System",
	// systemd's token for Hyper-V; the DMI vendor reads "Microsoft Corporation".
	"microsoft": "Hyper-V",
	// systemd's token for GCE; the DMI vendor reads "Google" / "Google Compute Engine".
	"google": "Google Compute Engine",
	// systemd's token for Apple Virtualization.framework. The longer keys above
	// cannot match it, because "apple" contains neither of them.
	"apple": "Apple Virtualization",
	// systemd's token for VirtualBox. The longest-match rule in mapHypervisor
	// keeps "Oracle VM VirtualBox" resolving to VirtualBox rather than racing
	// this key. Oracle Cloud cannot be caught by it: the whole Linux chain is
	// gated on the CPU hypervisor flag, so bare metal never reaches here, and
	// OCI VM shapes report QEMU.
	"oracle": "VirtualBox",
}

// hyper is a helper struct to avoid passing the connection and platform
// as function arguments.
type hyper struct {
	connection shared.Connection
	platform   *inventory.Platform
}

// Hypervisor returns the hypervisor of the system. Not cached here; the os.hypervisor
// resource memoizes it per connection via plugin.GetOrCompute.
func Hypervisor(conn shared.Connection, pf *inventory.Platform) (hypervisor string, ok bool) {
	if !pf.IsFamily(inventory.FAMILY_UNIX) && !pf.IsFamily(inventory.FAMILY_WINDOWS) {
		log.Debug().Msg("your platform is not supported for hypervisor detection")
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

	// Longest key first. Several vendor strings contain more than one key --
	// "Oracle VM VirtualBox" holds both "oracle" and "virtualbox" -- and Go
	// randomises map iteration, so picking whichever came first made the answer
	// differ between runs. The longest match is the most specific one.
	for _, key := range hypervisorKeysByLength() {
		if strings.Contains(info, key) {
			return knownHypervisors[key], true
		}
	}
	return "", false
}

var (
	hypervisorKeysOnce   sync.Once
	hypervisorKeysSorted []string
)

// hypervisorKeysByLength returns the knownHypervisors keys ordered longest
// first, so the most specific match wins and the result is reproducible.
func hypervisorKeysByLength() []string {
	hypervisorKeysOnce.Do(func() {
		hypervisorKeysSorted = make([]string, 0, len(knownHypervisors))
		for key := range knownHypervisors {
			hypervisorKeysSorted = append(hypervisorKeysSorted, key)
		}
		sort.Slice(hypervisorKeysSorted, func(i, j int) bool {
			if len(hypervisorKeysSorted[i]) != len(hypervisorKeysSorted[j]) {
				return len(hypervisorKeysSorted[i]) > len(hypervisorKeysSorted[j])
			}
			return hypervisorKeysSorted[i] < hypervisorKeysSorted[j]
		})
	})
	return hypervisorKeysSorted
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

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hypervisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// EC2 Nitro guests were reporting a null hypervisor: systemd-detect-virt says
// "amazon" and the DMI vendor fields say "Amazon EC2", neither of which had a
// mapping, so all three Linux detectors found their input and failed to map it.
//
// The value is the hypervisor, not the cloud: systemd emits "amazon" only for
// Nitro, while the older Xen instance families report "xen".
func TestMapHypervisorAmazonEC2(t *testing.T) {
	// systemd-detect-virt on a Nitro guest
	v, ok := mapHypervisor("amazon")
	assert.True(t, ok)
	assert.Equal(t, "AWS Nitro System", v)

	// DMI sys_vendor / board_vendor / bios_vendor
	v, ok = mapHypervisor("Amazon EC2")
	assert.True(t, ok)
	assert.Equal(t, "AWS Nitro System", v)

	// dmidecode output carries trailing whitespace and mixed case
	v, ok = mapHypervisor("  AMAZON EC2\n")
	assert.True(t, ok)
	assert.Equal(t, "AWS Nitro System", v)
}

// Older EC2 instances are Xen-based and must keep resolving to Xen, not be
// swallowed by the new entry.
func TestMapHypervisorXenStillWins(t *testing.T) {
	v, ok := mapHypervisor("xen")
	assert.True(t, ok)
	assert.Equal(t, "Xen", v)

	v, ok = mapHypervisor("Xen HVM domU")
	assert.True(t, ok)
	assert.Equal(t, "Xen", v)
}

// The existing mappings are unaffected, and an unknown vendor still reports
// nothing rather than guessing.
func TestMapHypervisorUnchanged(t *testing.T) {
	for input, want := range map[string]string{
		"kvm":         "KVM",
		"VMware, Inc": "VMware",
		"QEMU":        "QEMU",
		"hyper-v":     "Hyper-V",
	} {
		v, ok := mapHypervisor(input)
		assert.True(t, ok, input)
		assert.Equal(t, want, v, input)
	}

	_, ok := mapHypervisor("Some Unknown Vendor")
	assert.False(t, ok)
}

// systemd-detect-virt emits its own identifiers, which are not the product
// names. Without a mapping the guest is detected and then fails to map, so
// os.hypervisor reads null. Tokens taken from systemd src/basic/virt.c.
func TestMapHypervisorSystemdTokens(t *testing.T) {
	for token, want := range map[string]string{
		"amazon":    "AWS Nitro System",
		"microsoft": "Hyper-V",
		"google":    "Google Compute Engine",
		"apple":     "Apple Virtualization",
		"oracle":    "VirtualBox",
		// already mapped before this change, kept here so the whole systemd
		// vocabulary is covered in one place
		"kvm":       "KVM",
		"qemu":      "QEMU",
		"xen":       "Xen",
		"vmware":    "VMware",
		"parallels": "Parallels",
		"bhyve":     "bhyve",
		"powervm":   "IBM PowerVM",
	} {
		t.Run(token, func(t *testing.T) {
			v, ok := mapHypervisor(token)
			assert.True(t, ok, "systemd emits %q; it must map", token)
			assert.Equal(t, want, v)
		})
	}
}

// The DMI vendor strings for the same platforms must land on the same answer.
func TestMapHypervisorDMIVendors(t *testing.T) {
	for vendor, want := range map[string]string{
		"Amazon EC2":            "AWS Nitro System",
		"Microsoft Corporation": "Hyper-V",
		"Google":                "Google Compute Engine",
		"Google Compute Engine": "Google Compute Engine",
		"Apple Inc.":            "Apple Virtualization",
		"VMware, Inc.":          "VMware",
		"Xen":                   "Xen",
		"QEMU":                  "QEMU",
	} {
		t.Run(vendor, func(t *testing.T) {
			v, ok := mapHypervisor(vendor)
			assert.True(t, ok, vendor)
			assert.Equal(t, want, v)
		})
	}
}

// Several vendor strings contain more than one key. Go randomises map
// iteration, so before ordering by key length the answer differed between
// runs. The most specific match must win, every time.
func TestMapHypervisorLongestMatchWinsDeterministically(t *testing.T) {
	// contains both "oracle"-free "virtualbox" and, in the vendor form, "oracle"
	for i := 0; i < 50; i++ {
		v, ok := mapHypervisor("Oracle VM VirtualBox")
		assert.True(t, ok)
		assert.Equal(t, "VirtualBox", v, "longest key must win on every iteration")
	}

	// "apple virtual" is longer than "apple" and must take precedence
	for i := 0; i < 50; i++ {
		v, ok := mapHypervisor("Apple Virtual Machine")
		assert.True(t, ok)
		assert.Equal(t, "Apple Virtualization", v)
	}
}

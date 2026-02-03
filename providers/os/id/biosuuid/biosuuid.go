// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package biosuuid

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/smbios"
)

// BiosUUID returns the BIOS UUID (SMBIOS System UUID) for the platform.
// This is preferred over SerialNumber for VMs, as some hypervisors (e.g., OpenStack)
// pass through the host's serial number to VMs, making it non-unique.
// The BIOS UUID is typically unique per VM instance.
func BiosUUID(conn shared.Connection, p *inventory.Platform) (string, error) {
	mgr, err := smbios.ResolveManager(conn, p)
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform BIOS UUID")
	}
	if mgr == nil {
		return "", errors.New("cannot determine platform BIOS UUID")
	}

	info, err := mgr.Info()
	if err != nil {
		return "", errors.New("cannot determine platform BIOS UUID")
	}

	return info.SysInfo.UUID, nil
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package serialnumber

import (
	"github.com/pkg/errors"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/smbios"
)

func SerialNumber(conn shared.Connection, p *inventory.Platform) (string, error) {
	mgr, err := smbios.ResolveManager(conn, p)
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform serial number")
	}
	if mgr == nil {
		return "", errors.New("cannot determine platform serial number")
	}

	info, err := mgr.Info()
	if err != nil {
		return "", errors.New("cannot determine platform serial number")
	}

	return info.SysInfo.SerialNumber, nil
}

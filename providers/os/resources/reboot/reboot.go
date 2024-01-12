// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reboot

import (
	"errors"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

type Reboot interface {
	Name() string
	RebootPending() (bool, error)
}

func New(conn shared.Connection) (Reboot, error) {
	pf := conn.Asset().Platform

	switch {
	case pf.IsFamily("debian"):
		return &DebianReboot{conn: conn}, nil
	case pf.IsFamily("redhat") || pf.Name == "amazonlinux":
		return &RpmNewestKernel{conn: conn}, nil
	case pf.IsFamily(inventory.FAMILY_WINDOWS):
		return &WinReboot{conn: conn}, nil
	default:
		return nil, errors.New("your platform is not supported by reboot resource")
	}
}

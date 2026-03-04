// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

func GetWindowsHotpatch(conn shared.Connection, pf *inventory.Platform) (bool, error) {
	if !hotpatchSupported(pf) {
		return false, nil
	}
	return powershellGetWindowsHotpatch(conn, pf.Arch)
}

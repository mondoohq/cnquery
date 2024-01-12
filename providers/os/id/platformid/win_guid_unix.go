// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package platformid

import "go.mondoo.com/cnquery/v10/providers/os/connection/shared"

func windowsMachineId(conn shared.Connection) (string, error) {
	return PowershellWindowsMachineId(conn)
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package windows

import "go.mondoo.com/mql/v13/providers/os/connection/shared"

func GetWindowsESUStatus(conn shared.Connection) (*WindowsESUStatus, error) {
	return powershellGetWindowsESUStatus(conn)
}

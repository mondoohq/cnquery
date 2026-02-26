// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.com/mql/v13/providers/os/connection/shared"

func GetIntuneDeviceID(conn shared.Connection) (string, error) {
	return powershellGetIntuneDeviceID(conn)
}

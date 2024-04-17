// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import "go.mondoo.com/cnquery/v11/providers/os/connection/shared"

func GetWmiInformation(conn shared.Connection) (*WmicOSInformation, error) {
	return powershellGetWmiInformation(conn)
}

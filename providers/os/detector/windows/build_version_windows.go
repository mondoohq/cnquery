// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"runtime"

	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"golang.org/x/sys/windows/registry"
)

func GetWindowsOSBuild(conn shared.Connection) (*WindowsCurrentVersion, error) {
	// if we are running locally on windows, we want to avoid using powershell to be faster
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
		if err != nil {
			return nil, err
		}

		currentBuild, _, err := k.GetStringValue("CurrentBuild")
		if err != nil && err != registry.ErrNotExist {
			return nil, err
		}

		ubr, _, err := k.GetIntegerValue("UBR")
		if err != nil && err != registry.ErrNotExist {
			return nil, err
		}

		edition, _, err := k.GetStringValue("EditionID")
		if err != nil && err != registry.ErrNotExist {
			return nil, err
		}
		defer k.Close()

		return &WindowsCurrentVersion{
			CurrentBuild: currentBuild,
			EditionID:    edition,
			UBR:          int(ubr),
		}, nil
	}

	// for all non-local checks use powershell
	return powershellGetWindowsOSBuild(conn)
}

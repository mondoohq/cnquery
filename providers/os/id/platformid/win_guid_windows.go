// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package platformid

import (
	"errors"
	"runtime"

	wmi "github.com/StackExchange/wmi"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

func windowsMachineId(conn shared.Connection) (string, error) {
	// if we are running locally on windows, we want to avoid using powershell to be faster
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		// we always get a list or entries
		type win32ComputerSystemProduct struct {
			UUID *string
		}

		// query wmi to retrieve information
		var entries []win32ComputerSystemProduct
		if err := wmi.Query(wmiMachineIDQuery, &entries); err != nil {
			return "", err
		}

		if len(entries) != 1 || entries[0].UUID == nil {
			return "", errors.New("could not query machine id on windows")
		}

		return *entries[0].UUID, nil
	}

	return PowershellWindowsMachineId(conn)
}

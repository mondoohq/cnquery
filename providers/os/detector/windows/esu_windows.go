// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"runtime"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"golang.org/x/sys/windows/registry"
)

func GetWindowsESUStatus(conn shared.Connection) (*WindowsESUStatus, error) {
	log.Debug().Msg("checking Windows 10 ESU status")

	// if we are running locally on windows, check registry directly for subscription ESU
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		status := &WindowsESUStatus{}

		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\SoftwareProtectionPlatform\ESU`, registry.QUERY_VALUE)
		if err != nil {
			log.Debug().Err(err).Msg("could not open ESU registry key, ESU may not be configured")
			// Fall through to PowerShell for WMI-based MAK check
			return powershellGetWindowsESUStatus(conn)
		}
		defer k.Close()

		eligible, _, err := k.GetIntegerValue("Win10CommercialW365ESUEligible")
		if err != nil && err != registry.ErrNotExist {
			log.Debug().Err(err).Msg("could not get Win10CommercialW365ESUEligible value")
		}

		if eligible == 1 {
			status.SubscriptionEligible = true
			return status, nil
		}

		// Registry key exists but subscription not eligible, fall through to check MAK activation
		return powershellGetWindowsESUStatus(conn)
	}

	// for all non-local checks use powershell
	return powershellGetWindowsESUStatus(conn)
}

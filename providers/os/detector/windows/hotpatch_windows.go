// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"golang.org/x/sys/windows/registry"
)

func GetWindowsHotpatch(conn shared.Connection, pf *inventory.Platform) (bool, error) {
	log.Debug().Msg("checking windows hotpatch")

	buildNumber, err := strconv.Atoi(pf.Version)
	if err != nil {
		log.Error().Err(err).Msg("could not parse windows build number")
	}
	log.Debug().Int("buildNumber", buildNumber).Msg("parsed windows build number")
	if buildNumber < 20348 {
		return false, nil
	}
	// In case of Windows Server 2022+, check for hotpatching
	// This can be activated for on-prem or Azure Editions

	// if we are running locally on windows, we want to avoid using powershell to be faster
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Update\TargetingInfo\DynamicInstalled\Hotpatch.`+strings.ToLower(pf.Arch), registry.QUERY_VALUE)
		if err != nil {
			log.Debug().Err(err).Msg("could not open registry key DynamicInstalled")
			// Don't return an error here, as it is expected that this key may not exist
			return false, nil
		}
		defer k.Close()

		hotpatchName, _, err := k.GetStringValue("Name")
		if err != nil && err != registry.ErrNotExist {
			return false, err
		}

		systemKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\DeviceGuard`, registry.QUERY_VALUE)
		if err != nil {
			log.Debug().Err(err).Msg("could not open registry key DeviceGuard")
			// Don't return an error here, as it is expected that this key may not exist
			return false, nil
		}
		defer systemKey.Close()

		enableVirtualizationBasedSecurity, _, err := systemKey.GetIntegerValue("EnableVirtualizationBasedSecurity")
		if err != nil && err != registry.ErrNotExist {
			log.Debug().Err(err).Msg("could not get EnableVirtualizationBasedSecurity value")
			return false, err
		}

		memoryKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, registry.QUERY_VALUE)
		if err != nil {
			log.Debug().Err(err).Msg("could not open registry key Memory Management")
			// Don't return an error here, as it is expected that this key may not exist
			return false, nil
		}
		defer memoryKey.Close()

		hotPatchTableSize, _, err := memoryKey.GetIntegerValue("HotPatchTableSize")
		if err != nil && err != registry.ErrNotExist {
			log.Debug().Err(err).Msg("could not get HotPatchTableSize value")
			return false, err
		}

		log.Debug().Str("hotpatchName", hotpatchName).Int("enableVirtualizationBasedSecurity", int(enableVirtualizationBasedSecurity)).Int("hotPatchTableSize", int(hotPatchTableSize)).Msg("parsed windows hotpatch settings")

		return hotpatchName == HotpatchPackage && enableVirtualizationBasedSecurity == 1 && hotPatchTableSize > 0, nil
	}

	// for all non-local checks use powershell
	return powershellGetWindowsHotpatch(conn, pf.Arch)
}

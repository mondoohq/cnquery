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

func GetIntuneDeviceID(conn shared.Connection) (string, error) {
	log.Debug().Msg("checking Intune device ID")

	// if we are running locally on windows, we want to avoid using powershell to be faster
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		enrollmentsKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Enrollments`, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			log.Debug().Err(err).Msg("could not open Enrollments registry key")
			return "", nil
		}
		defer enrollmentsKey.Close()

		subkeys, err := enrollmentsKey.ReadSubKeyNames(-1)
		if err != nil {
			log.Debug().Err(err).Msg("could not read Enrollments subkeys")
			return "", nil
		}

		for _, subkey := range subkeys {
			dmClientPath := `SOFTWARE\Microsoft\Enrollments\` + subkey + `\DMClient\MS DM Server`
			dmClientKey, err := registry.OpenKey(registry.LOCAL_MACHINE, dmClientPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			entDMID, _, err := dmClientKey.GetStringValue("EntDMID")
			dmClientKey.Close()
			if err != nil {
				continue
			}

			if entDMID != "" {
				log.Debug().Str("EntDMID", entDMID).Msg("found Intune device ID")
				return entDMID, nil
			}
		}

		return "", nil
	}

	// for all non-local checks use powershell
	return powershellGetIntuneDeviceID(conn)
}

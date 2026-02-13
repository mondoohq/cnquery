// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build linux || darwin || netbsd || openbsd || freebsd
// +build linux darwin netbsd openbsd freebsd

package windows

import (
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

func GetWindowsHotpatch(conn shared.Connection, pf *inventory.Platform) (bool, error) {
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
	return powershellGetWindowsHotpatch(conn, pf.Arch)
}

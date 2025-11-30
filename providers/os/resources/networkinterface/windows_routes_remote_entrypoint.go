// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package networkinterface

import "github.com/rs/zerolog/log"

// List detects network routes on Windows (remote execution)
// Falls back to PowerShell and netstat since native APIs are not available
func (w *windowsRouteDetector) List() ([]Route, error) {
	routes, err := w.detectWindowsRoutesViaPowerShell()
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	log.Debug().Err(err).Int("routeCount", len(routes)).Msg("PowerShell Get-NetRoute failed or returned no routes, trying netstat")

	// fallback to netstat
	return w.detectWindowsRoutesViaNetstat()
}

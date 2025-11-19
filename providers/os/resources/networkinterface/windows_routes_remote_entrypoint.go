// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package networkinterface

import "github.com/rs/zerolog/log"

// detectWindowsRoutes detects network routes on Windows
func (n *netr) detectWindowsRoutes() ([]Route, error) {
	routes, err := n.detectWindowsRoutesViaPowerShell()
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	log.Debug().Err(err).Int("routeCount", len(routes)).Msg("PowerShell Get-NetRoute failed or returned no routes, trying netstat")

	// fallback to netstat
	return n.detectWindowsRoutesViaNetstat()
}

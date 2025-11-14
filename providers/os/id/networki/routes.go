// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// Routes returns the network routes of the system.
// This function dispatches to platform-specific implementations based on runtime detection.
func Routes(conn shared.Connection, pf *inventory.Platform) ([]Route, error) {
	n := &neti{conn, pf}

	if pf.IsFamily(inventory.FAMILY_LINUX) {
		return n.detectLinuxRoutes()
	}
	if pf.IsFamily(inventory.FAMILY_DARWIN) {
		return n.detectDarwinRoutes()
	}
	if pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return n.detectWindowsRoutes()
	}

	return nil, errors.New("your platform is not supported for the detection of network routes")
}

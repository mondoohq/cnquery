// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package groups

import (
	"errors"

	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

func ResolveManager(conn shared.Connection) (OSGroupManager, error) {
	var gm OSGroupManager

	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, errors.New("cannot find OS information for users detection")
	}

	// check darwin before unix since darwin is also a unix
	if asset.Platform.IsFamily("darwin") {
		gm = &OSXGroupManager{conn: conn}
	} else if asset.Platform.IsFamily("unix") {
		gm = &UnixGroupManager{conn: conn}
	} else if asset.Platform.IsFamily("windows") {
		gm = &WindowsGroupManager{conn: conn}
	}

	if gm == nil {
		return nil, errors.New("could not detect suitable group manager for platform: " + asset.Platform.Name)
	}

	return gm, nil
}

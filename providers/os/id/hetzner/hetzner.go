// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hetzner

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/id/hetznercompute"
)

func Detect(conn shared.Connection, pf *inventory.Platform) (string, string, []string) {
	mdsvc, err := hetznercompute.Resolve(conn, pf)
	if err != nil {
		log.Debug().Err(err).Msg("failed to get metadata resolver")
		return "", "", nil
	}
	id, err := mdsvc.Identify()
	if err != nil {
		log.Debug().Err(err).
			Strs("platform", pf.GetFamily()).
			Msg("failed to get Hetzner platform id")
		return "", "", nil
	}
	return id.InstanceID, id.InstanceName, nil
}

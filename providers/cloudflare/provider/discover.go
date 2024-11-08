// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package provider

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/resources"
)

func (s *Service) discover(conn *connection.CloudflareConnection) (*inventory.Inventory, error) {
	conf := conn.Asset().Connections[0]
	if conf.Discover == nil {
		return nil, nil
	}

	runtime, err := s.GetRuntime(conn.ID())
	if err != nil {
		return nil, err
	}

	return resources.Discover(runtime, conf.Options)
}

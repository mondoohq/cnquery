// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
)

func (s *Service) detect(asset *inventory.Asset, conn shared.Connection) error {
	asset.Platform = conn.Platform()

	return nil
}

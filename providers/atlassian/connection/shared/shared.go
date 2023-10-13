// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

type ConnectionType string

type Connection interface {
	ID() uint32
	Name() string
	Type() ConnectionType
	Asset() *inventory.Asset
	Host() string
}

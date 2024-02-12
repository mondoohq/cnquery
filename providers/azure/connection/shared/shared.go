// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

type ConnectionType string

type AzureConnection interface {
	ID() uint32
	ParentID() *uint32
	Name() string
	Type() ConnectionType
	Config() *inventory.Config
	Asset() *inventory.Asset
}

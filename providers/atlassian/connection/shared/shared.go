// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

type ConnectionType string

type Connection interface {
	plugin.Connection
	Name() string
	Type() ConnectionType
	Asset() *inventory.Asset
	PlatformInfo() *inventory.Platform
	PlatformID() string
	Config() *inventory.Config
}

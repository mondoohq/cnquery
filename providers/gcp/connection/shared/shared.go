// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

type ConnectionType string

type GcpConnection interface {
	plugin.Connection
	Name() string
	Type() ConnectionType
	Config() *inventory.Config
	Asset() *inventory.Asset
}

type Capabilities byte

const (
	Capability_Aws_Ebs Capabilities = 1 << iota
)

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

type ConnectionType string

// References:
// - https://www.terraform.io/docs/language/syntax/configuration.html
// - https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md
type Connection struct {
	id              uint32
	name            string
	asset           *inventory.Asset
	platformID      string
	assetType       terraformAssetType
	parsed          *hclparse.Parser
	tfVars          map[string]*hcl.Attribute
	modulesManifest *ModuleManifest
	state           *State
	plan            *Plan
	closer          func()
}

func (c *Connection) Close() {
	if c.closer != nil {
		c.closer()
	}
}

func (c *Connection) Kind() string {
	return "code"
}

func (c *Connection) Runtime() string {
	return "terraform"
}

func (c *Connection) Asset() *inventory.Asset {
	return c.asset
}

func (c *Connection) ID() uint32 {
	return c.id
}

func (c *Connection) Name() string {
	return c.name
}

func (c *Connection) Parser() *hclparse.Parser {
	return c.parsed
}

func (c *Connection) TfVars() map[string]*hcl.Attribute {
	return c.tfVars
}

func (c *Connection) ModulesManifest() *ModuleManifest {
	return c.modulesManifest
}

func (c *Connection) Identifier() (string, error) {
	return c.platformID, nil
}

func (c *Connection) State() (*State, error) {
	return c.state, nil
}

func (c *Connection) Plan() (*Plan, error) {
	return c.plan, nil
}

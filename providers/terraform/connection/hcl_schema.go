// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import "github.com/hashicorp/hcl/v2"

// TODO: update the schema
// Same schema as defined in terraform itself https://github.com/hashicorp/terraform/blob/60ab95d3a7cde5e2aca82ee9df78e57cbba04534/internal/configs/parser_config.go#L269
var TerraformSchema_1 = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "terraform",
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
		{
			Type: "locals",
		},
		{
			Type:       "output",
			LabelNames: []string{"name"},
		},
		{
			Type:       "module",
			LabelNames: []string{"name"},
		},
		{
			Type:       "resource",
			LabelNames: []string{"type", "name"},
		},
		{
			Type:       "data",
			LabelNames: []string{"type", "name"},
		},
		{
			Type: "moved",
		},
		{
			Type: "removed",
		},
		{
			Type: "import",
		},
		{
			Type:       "check",
			LabelNames: []string{"name"},
		},
	},
}

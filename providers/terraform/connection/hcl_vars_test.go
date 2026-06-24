// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tfVarsFromString(t *testing.T, src string) map[string]*hcl.Attribute {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(src), "test.tfvars", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "tfvars parse errors: %s", diags.Error())
	attrs, diags := f.Body.JustAttributes()
	require.False(t, diags.HasErrors(), "tfvars attribute errors: %s", diags.Error())
	return attrs
}

func TestVariableEvalContext(t *testing.T) {
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL([]byte(`
variable "bucket_acl" { default = "public-read" }
variable "env"        { default = "prod" }
variable "no_default" { type = string }

locals {
  acl   = var.env == "prod" ? "private" : "public-read"
  chain = local.acl
}
`), "main.tf")
	require.False(t, diags.HasErrors(), "hcl parse errors: %s", diags.Error())

	// .tfvars overrides the variable default (Terraform precedence).
	c := &Connection{
		parsed: parser,
		tfVars: tfVarsFromString(t, `bucket_acl = "log-delivery-write"`),
	}

	ctx := c.VariableEvalContext()
	require.NotNil(t, ctx)

	vars := ctx.Variables["var"]
	require.False(t, vars.IsNull())
	assert.Equal(t, "log-delivery-write", vars.GetAttr("bucket_acl").AsString(), "tfvars should override the default")
	assert.Equal(t, "prod", vars.GetAttr("env").AsString(), "default should be used when no tfvars value")
	// a variable without a default and without a tfvars value is absent.
	assert.False(t, vars.Type().HasAttribute("no_default"))

	locals := ctx.Variables["local"]
	require.False(t, locals.IsNull())
	assert.Equal(t, "private", locals.GetAttr("acl").AsString(), "local should resolve via var")
	assert.Equal(t, "private", locals.GetAttr("chain").AsString(), "local-referencing-local should resolve via fixpoint")

	// memoized: a second call returns the same context.
	assert.Same(t, ctx, c.VariableEvalContext())
}

// TestVariableEvalContext_NoParser ensures plan/state assets (no HCL parser)
// produce a nil context rather than panicking.
func TestVariableEvalContext_NoParser(t *testing.T) {
	c := &Connection{}
	assert.Nil(t, c.VariableEvalContext())
}

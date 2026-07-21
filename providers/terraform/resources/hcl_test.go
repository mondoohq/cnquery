// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/connection"
)

func blockWithLabels(labels ...string) *mqlTerraformBlock {
	data := make([]any, len(labels))
	for i, l := range labels {
		data[i] = l
	}
	b := &mqlTerraformBlock{}
	b.Labels = plugin.TValue[[]any]{Data: data, State: plugin.StateIsSet}
	return b
}

func TestBlockResourceTypeAndName(t *testing.T) {
	// resource "aws_instance" "web" { ... }
	b := blockWithLabels("aws_instance", "web")
	rt, err := b.resourceType()
	require.NoError(t, err)
	assert.Equal(t, "aws_instance", rt)
	rn, err := b.resourceName()
	require.NoError(t, err)
	assert.Equal(t, "web", rn)

	// resourceType mirrors nameLabel (first label)
	nl, err := b.nameLabel()
	require.NoError(t, err)
	assert.Equal(t, nl, rt)

	// Blocks without a second label (e.g. provider blocks) yield an empty name.
	single := blockWithLabels("aws")
	rn, err = single.resourceName()
	require.NoError(t, err)
	assert.Equal(t, "", rn)
	rt, err = single.resourceType()
	require.NoError(t, err)
	assert.Equal(t, "aws", rt)

	// Blocks with no labels yield empty strings rather than panicking.
	none := blockWithLabels()
	rt, err = none.resourceType()
	require.NoError(t, err)
	assert.Equal(t, "", rt)
	rn, err = none.resourceName()
	require.NoError(t, err)
	assert.Equal(t, "", rn)
}

// TestHclResources_NoParser_DoNotPanic is a regression test for
// https://github.com/mondoohq/mql/issues/2970: the terraform.files,
// terraform.file.blocks and terraform.module.block resources used to
// dereference a nil *hclparse.Parser and crash the provider on plan and
// state assets (which have no HCL parser). They must now return empty
// results instead of panicking.
func TestHclResources_NoParser_DoNotPanic(t *testing.T) {
	// A zero-value Connection mirrors what NewPlanConnection / NewStateConnection
	// produce: no HCL parser is set, so Parser() returns nil.
	runtime := &plugin.Runtime{Connection: &connection.Connection{}}

	t.Run("terraform.files", func(t *testing.T) {
		tf := &mqlTerraform{}
		tf.MqlRuntime = runtime
		files, err := tf.files()
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("terraform.file.blocks", func(t *testing.T) {
		file := &mqlTerraformFile{}
		file.MqlRuntime = runtime
		file.Path = plugin.TValue[string]{Data: "main.tf", State: plugin.StateIsSet}
		blocks, err := file.blocks()
		require.NoError(t, err)
		assert.Empty(t, blocks)
	})

	t.Run("terraform.module.block", func(t *testing.T) {
		module := &mqlTerraformModule{}
		module.MqlRuntime = runtime
		module.Key = plugin.TValue[string]{Data: "vpc", State: plugin.StateIsSet}
		block, err := module.block()
		require.NoError(t, err)
		assert.Nil(t, block)
		// The accessor must mark the field resolved-and-null so the runtime
		// does not re-fetch indefinitely.
		assert.True(t, module.Block.State&plugin.StateIsNull != 0)
	})
}

// parseAttrs parses an HCL snippet and returns the top-level attributes.
// The snippet must contain attribute definitions only (no blocks).
func parseAttrs(t *testing.T, src string) map[string]*hcl.Attribute {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse errors: %s", diags.Error())
	attrs, diags := f.Body.JustAttributes()
	require.False(t, diags.HasErrors(), "attribute errors: %s", diags.Error())
	return attrs
}

// containsString reports whether v (which may be a string, []any, or nested
// list) contains the given string anywhere in its tree.
func containsString(v any, target string) bool {
	switch x := v.(type) {
	case string:
		return x == target
	case []any:
		for _, item := range x {
			if containsString(item, target) {
				return true
			}
		}
	case map[string]any:
		for _, item := range x {
			if containsString(item, target) {
				return true
			}
		}
	}
	return false
}

// TestGetCtyValue_ForExpr_Tuple verifies that tuple-form for-expressions
// like `[for sg in data.x : sg.id]` return the references they contain
// instead of triggering an "unknown type *hclsyntax.ForExpr" warning and
// nil result.
func TestGetCtyValue_ForExpr_Tuple(t *testing.T) {
	attrs := parseAttrs(t, `vpc_security_group_ids = [for sg in data.aws_security_group.ec2 : sg.id]`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["vpc_security_group_ids"]
	require.NotNil(t, v, "for-expression should not return nil")
	assert.True(t, containsString(v, "data.aws_security_group.ec2"),
		"result should reference data.aws_security_group.ec2, got: %#v", v)
}

// TestGetCtyValue_ForExpr_Object verifies that object-form for-expressions
// like `{ for k in coll : k => f(k) }` return a map with the references they
// contain.
func TestGetCtyValue_ForExpr_Object(t *testing.T) {
	attrs := parseAttrs(t, `
subnet_id_by_az_suffix = {
  for zone in ["a", "b"] :
  zone => data.aws_subnet.ec2[zone].id
}
`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["subnet_id_by_az_suffix"]
	require.NotNil(t, v, "object for-expression should not return nil")
	assert.True(t, containsString(v, "data.aws_subnet.ec2"),
		"result should reference data.aws_subnet.ec2, got: %#v", v)
}

// TestGetCtyValue_ConditionalExpr_UnboundVars verifies that ternaries
// involving unbound variables return the references in the branches instead
// of an empty list.
func TestGetCtyValue_ConditionalExpr_UnboundVars(t *testing.T) {
	attrs := parseAttrs(t, `ami_id = var.disaster_recovery_mode ? var.disaster_recovery_ami_id : data.aws_ami.shared_image.id`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["ami_id"]
	require.NotNil(t, v, "conditional should not return nil")
	assert.True(t, containsString(v, "var.disaster_recovery_mode"),
		"result should reference var.disaster_recovery_mode, got: %#v", v)
	assert.True(t, containsString(v, "var.disaster_recovery_ami_id"),
		"result should reference var.disaster_recovery_ami_id, got: %#v", v)
	assert.True(t, containsString(v, "data.aws_ami.shared_image.id"),
		"result should reference data.aws_ami.shared_image.id, got: %#v", v)
}

// TestGetCtyValue_NegativeNumber is a regression test for #9080: a negative
// numeric literal parses as a unary-negate op wrapping a number literal, and
// must retain its sign instead of collapsing to the positive magnitude (which
// broke checks comparing against a negative sentinel, e.g. Lambda's `-1`
// "unreserved" value or a security-group protocol of `-1`).
func TestGetCtyValue_NegativeNumber(t *testing.T) {
	attrs := parseAttrs(t, `reserved_concurrent_executions = -1`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)
	assert.Equal(t, float64(-1), got["reserved_concurrent_executions"])
}

// TestGetCtyValue_LogicalNot verifies the other unary operator (`!`) is applied
// rather than dropped.
func TestGetCtyValue_LogicalNot(t *testing.T) {
	attrs := parseAttrs(t, `disabled = !true`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)
	assert.Equal(t, false, got["disabled"])
}

// TestGetCtyValue_UnaryOp_UnboundOperand verifies a unary op over an unbound
// operand (e.g. `-var.x`) still surfaces the operand's references instead of
// failing, preserving the reference-surfacing fallback.
func TestGetCtyValue_UnaryOp_UnboundOperand(t *testing.T) {
	attrs := parseAttrs(t, `capacity = -var.desired_capacity`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)
	assert.True(t, containsString(got["capacity"], "var.desired_capacity"),
		"result should reference var.desired_capacity, got: %#v", got["capacity"])
}

// resolvingCtx builds an eval context with resolved var.*/local.* values,
// mimicking what Connection.VariableEvalContext produces when the
// TerraformResolveVars feature flag is active.
func resolvingCtx(vars, locals map[string]cty.Value) *hcl.EvalContext {
	ctx := &hcl.EvalContext{Functions: hclFunctions(), Variables: map[string]cty.Value{}}
	if len(vars) > 0 {
		ctx.Variables["var"] = cty.ObjectVal(vars)
	}
	if len(locals) > 0 {
		ctx.Variables["local"] = cty.ObjectVal(locals)
	}
	return ctx
}

// TestResolveVariables_Scalar verifies that a var.* reference resolves to the
// variable's effective value when the resolving context is supplied.
func TestResolveVariables_Scalar(t *testing.T) {
	attrs := parseAttrs(t, `acl = var.bucket_acl`)
	ctx := resolvingCtx(map[string]cty.Value{"bucket_acl": cty.StringVal("public-read")}, nil)
	got, err := hclResolvedAttributesToDict(attrs, ctx)
	require.NoError(t, err)
	assert.Equal(t, "public-read", got["acl"])
}

// TestResolveVariables_Local verifies local.* references resolve too.
func TestResolveVariables_Local(t *testing.T) {
	attrs := parseAttrs(t, `acl = local.acl`)
	ctx := resolvingCtx(nil, map[string]cty.Value{"acl": cty.StringVal("private")})
	got, err := hclResolvedAttributesToDict(attrs, ctx)
	require.NoError(t, err)
	assert.Equal(t, "private", got["acl"])
}

// TestResolveVariables_Types verifies non-string cty values convert to the
// expected Go types (bool, number, list).
func TestResolveVariables_Types(t *testing.T) {
	ctx := resolvingCtx(map[string]cty.Value{
		"enabled": cty.False,
		"port":    cty.NumberIntVal(22),
		"cidrs":   cty.ListVal([]cty.Value{cty.StringVal("0.0.0.0/0")}),
		"tags":    cty.ObjectVal(map[string]cty.Value{"env": cty.StringVal("prod")}),
	}, nil)
	attrs := parseAttrs(t, `
encrypted   = var.enabled
from_port   = var.port
cidr_blocks = var.cidrs
env         = var.tags.env
`)
	got, err := hclResolvedAttributesToDict(attrs, ctx)
	require.NoError(t, err)
	assert.Equal(t, false, got["encrypted"])
	assert.Equal(t, float64(22), got["from_port"])
	assert.Equal(t, []any{"0.0.0.0/0"}, got["cidr_blocks"])
	assert.Equal(t, "prod", got["env"], "nested attribute access should resolve")
}

// TestResolveVariables_Fallback verifies that an unresolvable reference (no
// such variable, or a data/resource ref) still falls back to the reference
// string even when a resolving context is supplied.
func TestResolveVariables_Fallback(t *testing.T) {
	ctx := resolvingCtx(map[string]cty.Value{"bucket_acl": cty.StringVal("public-read")}, nil)

	missing := parseAttrs(t, `acl = var.missing`)
	got, err := hclResolvedAttributesToDict(missing, ctx)
	require.NoError(t, err)
	assert.Equal(t, "var.missing", got["acl"], "unknown var should fall back to reference string")

	dataRef := parseAttrs(t, `ami = data.aws_ami.x.id`)
	got, err = hclResolvedAttributesToDict(dataRef, ctx)
	require.NoError(t, err)
	assert.True(t, containsString(got["ami"], "data.aws_ami.x.id"),
		"data references must still surface as a reference string, got: %#v", got["ami"])
}

// TestResolveVariables_TemplateInterpolation verifies a var embedded in a
// string template resolves the whole string.
func TestResolveVariables_TemplateInterpolation(t *testing.T) {
	ctx := resolvingCtx(map[string]cty.Value{"env": cty.StringVal("prod")}, nil)
	attrs := parseAttrs(t, `name = "app-${var.env}-bucket"`)
	got, err := hclResolvedAttributesToDict(attrs, ctx)
	require.NoError(t, err)
	assert.Equal(t, "app-prod-bucket", got["name"])
}

// TestGetCtyValue_IndexExpr verifies that index expressions
// like `m[k]` traverse into both the collection and the key.
func TestGetCtyValue_IndexExpr(t *testing.T) {
	attrs := parseAttrs(t, `subnet_id = local.subnet_id_by_az_suffix[local.az_suffix]`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["subnet_id"]
	require.NotNil(t, v, "index expression should not return nil")
	assert.True(t, containsString(v, "local.subnet_id_by_az_suffix"),
		"result should reference local.subnet_id_by_az_suffix, got: %#v", v)
}

// TestGetCtyValue_BinaryOp verifies binary comparisons
// like `var.x == "y"` surface their operand references.
func TestGetCtyValue_BinaryOp(t *testing.T) {
	attrs := parseAttrs(t, `match = var.availability_zone == "account_based"`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["match"]
	require.NotNil(t, v, "binary op expression should not return nil")
	assert.True(t, containsString(v, "var.availability_zone"),
		"result should reference var.availability_zone, got: %#v", v)
}

// TestGetCtyValue_RelativeTraversal verifies that relative traversals
// (e.g. result of an index/splat followed by `.field`) flow through.
func TestGetCtyValue_RelativeTraversal(t *testing.T) {
	attrs := parseAttrs(t, `first_id = random_shuffle.ec2.result[0]`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["first_id"]
	require.NotNil(t, v, "relative traversal should not return nil")
	assert.True(t, containsString(v, "random_shuffle.ec2.result"),
		"result should reference random_shuffle.ec2.result, got: %#v", v)
}

// TestGetCtyValue_SplatExpr verifies that splat expressions like
// `data.aws_instances.all[*].id` surface the source reference.
func TestGetCtyValue_SplatExpr(t *testing.T) {
	attrs := parseAttrs(t, `instance_ids = data.aws_instances.all[*].id`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)

	v := got["instance_ids"]
	require.NotNil(t, v, "splat expression should not return nil")
	assert.True(t, containsString(v, "data.aws_instances.all"),
		"result should reference data.aws_instances.all, got: %#v", v)
}

// TestGetCtyValue_StaticTernaryStillEvaluates verifies that conditionals
// that *can* be evaluated to a string still resolve the same way as before
// — we only fall back to reference collection when evaluation fails.
func TestGetCtyValue_StaticTernaryStillEvaluates(t *testing.T) {
	attrs := parseAttrs(t, `pick = true ? "yes" : "no"`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)
	assert.True(t, containsString(got["pick"], "yes"),
		"static ternary should evaluate to 'yes', got: %#v", got["pick"])
}

// TestGetCtyValue_BoolTernary_Resolved is a regression test for #9078: a
// ternary that resolves to a non-string (the natural `... ? true : false`
// output) used to fall through to reference-surfacing and return the branch
// list instead of the scalar, so a `!= true` scalar-equality check never
// matched. With var defaults in the context it must resolve to the scalar.
func TestGetCtyValue_BoolTernary_Resolved(t *testing.T) {
	attrs := parseAttrs(t, `privileged_mode = var.privileged ? true : false`)
	ctx := resolvingCtx(map[string]cty.Value{"privileged": cty.True}, nil)
	got, err := hclResolvedAttributesToDict(attrs, ctx)
	require.NoError(t, err)
	assert.Equal(t, true, got["privileged_mode"])
}

// TestGetCtyValue_StaticBoolTernary verifies a statically-knowable bool ternary
// resolves to the scalar even without any variable context.
func TestGetCtyValue_StaticBoolTernary(t *testing.T) {
	attrs := parseAttrs(t, `enabled = true ? false : true`)
	got, err := hclResolvedAttributesToDict(attrs, nil)
	require.NoError(t, err)
	assert.Equal(t, false, got["enabled"])
}

// TestHclAttributesToDict_NoUnknownTypeWarning regression-tests the customer
// case: parsing a file with for-expressions, conditionals and index
// expressions must not panic, must not return nil for any attribute, and
// must surface the references inside.
func TestHclAttributesToDict_CustomerFile(t *testing.T) {
	src := `
locals {
  subnet_id_by_az_suffix = {
    for zone in ["a", "b"] :
    zone => one([for subnet_id in data.aws_subnets.ec2.ids : subnet_id if endswith(data.aws_subnet.ec2[subnet_id].availability_zone, zone)])
  }
  ami_id = var.disaster_recovery_mode ? var.disaster_recovery_ami_id : data.aws_ami.shared_image.id
}
`
	f, diags := hclsyntax.ParseConfig([]byte(src), "customer.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), diags.Error())

	body, ok := f.Body.(*hclsyntax.Body)
	require.True(t, ok)
	require.Len(t, body.Blocks, 1)
	localsBlock := body.Blocks[0]

	hclAttrs := map[string]*hcl.Attribute{}
	for name, a := range localsBlock.Body.Attributes {
		hclAttrs[name] = a.AsHCLAttribute()
	}
	dict, err := hclAttributesToDict(hclAttrs, nil)
	require.NoError(t, err)

	require.NotNil(t, dict["subnet_id_by_az_suffix"])
	require.NotNil(t, dict["ami_id"])

	amiVal := dict["ami_id"].(map[string]any)["value"]
	assert.True(t, containsString(amiVal, "var.disaster_recovery_mode"),
		"ami_id should reference var.disaster_recovery_mode, got: %#v", amiVal)
}

// resourceBody parses an HCL snippet containing a single resource block and
// returns that block's body, so tests can exercise nested-block flattening.
func resourceBody(t *testing.T, src string) *hclsyntax.Body {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse errors: %s", diags.Error())
	body, ok := f.Body.(*hclsyntax.Body)
	require.True(t, ok)
	require.Len(t, body.Blocks, 1, "snippet must contain exactly one top-level block")
	return body.Blocks[0].Body
}

// TestHclBodyToValuesDict_FlattensNestedBlocksToPlanStateShape verifies that
// values() folds child blocks into the arguments dict as lists-of-maps keyed
// by block type — the same shape Terraform plan (change.after) and state
// (values) expose — so a single MQL body can run against all three backends.
func TestHclBodyToValuesDict_FlattensNestedBlocksToPlanStateShape(t *testing.T) {
	// resource "aws_eks_cluster" "example" { ... }
	body := resourceBody(t, `
resource "aws_eks_cluster" "example" {
  name = "example"

  encryption_config {
    resources = ["secrets"]
    provider {
      key_arn = "arn:aws:kms:us-east-1:111:key/abc"
    }
  }

  vpc_config {
    endpoint_private_access = true
    endpoint_public_access  = false
  }
}
`)

	values, err := hclBodyToValuesDict(body, nil)
	require.NoError(t, err)

	// Scalar arguments stay as direct values.
	assert.Equal(t, "example", values["name"])

	// A nested block becomes a []any of maps keyed by the block type — even
	// when it appears once — matching the plan/state JSON representation.
	encList, ok := values["encryption_config"].([]any)
	require.True(t, ok, "encryption_config should be []any, got %#v", values["encryption_config"])
	require.Len(t, encList, 1)
	enc := encList[0].(map[string]any)

	// Nested-within-nested blocks flatten recursively the same way.
	provList, ok := enc["provider"].([]any)
	require.True(t, ok, "provider should be []any, got %#v", enc["provider"])
	require.Len(t, provList, 1)
	assert.Equal(t, "arn:aws:kms:us-east-1:111:key/abc", provList[0].(map[string]any)["key_arn"])

	// Booleans round-trip as bools so `_['endpoint_public_access'] == false` works.
	vpcList, ok := values["vpc_config"].([]any)
	require.True(t, ok, "vpc_config should be []any, got %#v", values["vpc_config"])
	require.Len(t, vpcList, 1)
	assert.Equal(t, true, vpcList[0].(map[string]any)["endpoint_private_access"])
	assert.Equal(t, false, vpcList[0].(map[string]any)["endpoint_public_access"])
}

// TestHclBodyToValuesDict_RepeatedBlocksBecomeList verifies that repeated
// nested blocks of the same type (e.g. multiple database_flags) collect into
// a single list, matching how plan/state represent them.
func TestHclBodyToValuesDict_RepeatedBlocksBecomeList(t *testing.T) {
	body := resourceBody(t, `
resource "google_sql_database_instance" "example" {
  settings {
    database_flags {
      name  = "skip_show_database"
      value = "on"
    }
    database_flags {
      name  = "local_infile"
      value = "off"
    }
  }
}
`)

	values, err := hclBodyToValuesDict(body, nil)
	require.NoError(t, err)

	settings, ok := values["settings"].([]any)
	require.True(t, ok)
	require.Len(t, settings, 1)

	flags, ok := settings[0].(map[string]any)["database_flags"].([]any)
	require.True(t, ok, "database_flags should be []any, got %#v", settings[0].(map[string]any)["database_flags"])
	require.Len(t, flags, 2)
	assert.Equal(t, "skip_show_database", flags[0].(map[string]any)["name"])
	assert.Equal(t, "local_infile", flags[1].(map[string]any)["name"])
}

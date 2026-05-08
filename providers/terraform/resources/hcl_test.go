// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	got, err := hclResolvedAttributesToDict(attrs)
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
	got, err := hclResolvedAttributesToDict(attrs)
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
	got, err := hclResolvedAttributesToDict(attrs)
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

// TestGetCtyValue_IndexExpr verifies that index expressions
// like `m[k]` traverse into both the collection and the key.
func TestGetCtyValue_IndexExpr(t *testing.T) {
	attrs := parseAttrs(t, `subnet_id = local.subnet_id_by_az_suffix[local.az_suffix]`)
	got, err := hclResolvedAttributesToDict(attrs)
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
	got, err := hclResolvedAttributesToDict(attrs)
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
	got, err := hclResolvedAttributesToDict(attrs)
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
	got, err := hclResolvedAttributesToDict(attrs)
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
	got, err := hclResolvedAttributesToDict(attrs)
	require.NoError(t, err)
	assert.True(t, containsString(got["pick"], "yes"),
		"static ternary should evaluate to 'yes', got: %#v", got["pick"])
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
	dict, err := hclAttributesToDict(hclAttrs)
	require.NoError(t, err)

	require.NotNil(t, dict["subnet_id_by_az_suffix"])
	require.NotNil(t, dict["ami_id"])

	amiVal := dict["ami_id"].(map[string]any)["value"]
	assert.True(t, containsString(amiVal, "var.disaster_recovery_mode"),
		"ami_id should reference var.disaster_recovery_mode, got: %#v", amiVal)
}

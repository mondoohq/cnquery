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

// parseTopBlocks parses a snippet and returns its top-level *hcl.Blocks
// keyed by `type.name` (or `type.label1.label2` for resources / data).
func parseTopBlocks(t *testing.T, src string) (map[string]*hcl.Block, *hcl.File) {
	t.Helper()
	file, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse errors: %s", diags.Error())

	body, ok := file.Body.(*hclsyntax.Body)
	require.True(t, ok)

	out := map[string]*hcl.Block{}
	for _, b := range body.Blocks {
		hb := syntaxToHclBlock(b)
		key := b.Type
		for _, l := range b.Labels {
			key += "." + l
		}
		out[key] = hb
	}
	return out, file
}

func TestProviderFromType(t *testing.T) {
	assert.Equal(t, "aws", providerFromType("aws_s3_bucket"))
	assert.Equal(t, "google", providerFromType("google_compute_instance"))
	assert.Equal(t, "", providerFromType("nodash"))
	assert.Equal(t, "", providerFromType(""))
	// leading underscore: index 0, not > 0, so empty
	assert.Equal(t, "", providerFromType("_leading"))
}

func TestReadStringAttr_Defaults(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
variable "x" {
  description = "hello"
  type        = string
}
`)
	v := blocks["variable.x"]
	require.NotNil(t, v)
	assert.Equal(t, "hello", readStringAttr(v, "description"))
	// missing attribute -> empty string
	assert.Equal(t, "", readStringAttr(v, "default"))
	// non-string attribute (`type = string` is a type expression) -> empty
	assert.Equal(t, "", readStringAttr(v, "type"))
}

func TestReadBoolAttr_Default(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
variable "x" {
  sensitive = true
}
`)
	v := blocks["variable.x"]
	assert.True(t, readBoolAttr(v, "sensitive", false))
	assert.False(t, readBoolAttr(v, "missing", false))
	assert.True(t, readBoolAttr(v, "missing", true), "default value should be returned")
}

func TestAttributeDict_Excludes(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
provider "aws" {
  alias   = "west"
  version = "~> 5.0"
  region  = "us-west-2"
  profile = "prod"
}
`)
	p := blocks["provider.aws"]
	dict, err := attributeDict(p, "alias", "version")
	require.NoError(t, err)
	assert.Contains(t, dict, "region")
	assert.Contains(t, dict, "profile")
	assert.NotContains(t, dict, "alias")
	assert.NotContains(t, dict, "version")
}

func TestTraversalToString(t *testing.T) {
	src := `depends_on = [aws_s3_bucket.logs, module.foo.bar]`
	attrs := parseAttrs(t, src)
	attr := attrs["depends_on"]
	tuple, ok := attr.Expr.(*hclsyntax.TupleConsExpr)
	require.True(t, ok)
	require.Len(t, tuple.Exprs, 2)

	assert.Equal(t, "aws_s3_bucket.logs", traversalToString(tuple.Exprs[0]))
	assert.Equal(t, "module.foo.bar", traversalToString(tuple.Exprs[1]))
}

func TestTupleToRefStrings(t *testing.T) {
	src := `depends_on = [aws_s3_bucket.logs, aws_iam_role.execute]`
	attrs := parseAttrs(t, src)
	got := tupleToRefStrings(attrs["depends_on"])
	require.Len(t, got, 2)
	assert.Equal(t, "aws_s3_bucket.logs", got[0])
	assert.Equal(t, "aws_iam_role.execute", got[1])

	// nil attribute -> empty slice (not nil) so callers can rely on len()
	got = tupleToRefStrings(nil)
	assert.NotNil(t, got)
	assert.Empty(t, got)

	// non-tuple attribute -> empty slice
	scalarAttrs := parseAttrs(t, `depends_on = "just a string"`)
	got = tupleToRefStrings(scalarAttrs["depends_on"])
	assert.Empty(t, got)
}

func TestParseIgnoreChanges_All(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
resource "aws_s3_bucket" "logs" {
  lifecycle {
    ignore_changes = all
  }
}
`)
	r := blocks["resource.aws_s3_bucket.logs"]
	lc := findChildBlock(r, "lifecycle")
	require.NotNil(t, lc)
	got := parseIgnoreChanges(lc)
	require.Len(t, got, 1)
	assert.Equal(t, "all", got[0])
}

func TestParseIgnoreChanges_List(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
resource "aws_s3_bucket" "logs" {
  lifecycle {
    ignore_changes = [tags, versioning, lifecycle_rule]
  }
}
`)
	r := blocks["resource.aws_s3_bucket.logs"]
	lc := findChildBlock(r, "lifecycle")
	require.NotNil(t, lc)
	got := parseIgnoreChanges(lc)
	require.Len(t, got, 3)
	assert.Equal(t, "tags", got[0])
	assert.Equal(t, "versioning", got[1])
	assert.Equal(t, "lifecycle_rule", got[2])
}

func TestParseIgnoreChanges_Missing(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
resource "aws_s3_bucket" "logs" {
  lifecycle {
    prevent_destroy = true
  }
}
`)
	r := blocks["resource.aws_s3_bucket.logs"]
	lc := findChildBlock(r, "lifecycle")
	require.NotNil(t, lc)
	got := parseIgnoreChanges(lc)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestFindChildBlock(t *testing.T) {
	blocks, _ := parseTopBlocks(t, `
resource "aws_s3_bucket" "logs" {
  lifecycle {
    prevent_destroy = true
  }
}
`)
	r := blocks["resource.aws_s3_bucket.logs"]
	lc := findChildBlock(r, "lifecycle")
	require.NotNil(t, lc)
	assert.Equal(t, "lifecycle", lc.Type)

	// non-existent child
	missing := findChildBlock(r, "validation")
	assert.Nil(t, missing)

	// nil parent
	assert.Nil(t, findChildBlock(nil, "lifecycle"))
}

func TestSyntaxToHclBlock_PreservesLabels(t *testing.T) {
	file, diags := hclsyntax.ParseConfig([]byte(`
resource "aws_instance" "web" {
  ami = "ami-1"
}
`), "t.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors())
	body := file.Body.(*hclsyntax.Body)
	require.Len(t, body.Blocks, 1)

	hb := syntaxToHclBlock(body.Blocks[0])
	assert.Equal(t, "resource", hb.Type)
	assert.Equal(t, []string{"aws_instance", "web"}, hb.Labels)
	require.NotNil(t, hb.Body)
}

func TestSortedAttrKeys(t *testing.T) {
	src := `
zeta   = 1
alpha  = 2
middle = 3
`
	attrs := parseAttrs(t, src)
	got := sortedAttrKeys(attrs)
	assert.Equal(t, []string{"alpha", "middle", "zeta"}, got)
}

func TestResourceMetaArgsExcludedFromArguments(t *testing.T) {
	// Verifies the meta-arg exclusion set is wired correctly so
	// terraform.resource.arguments() doesn't leak meta-arguments.
	for _, k := range []string{"count", "for_each", "depends_on", "provider"} {
		_, present := resourceMetaArgs[k]
		assert.True(t, present, "%s should be in resourceMetaArgs", k)
	}
	// Sanity: a real argument is not in the exclusion set
	_, present := resourceMetaArgs["bucket"]
	assert.False(t, present)
}

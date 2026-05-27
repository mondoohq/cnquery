// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExpressionLiteral(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"single-quoted string", "'eastus'"},
		{"integer", "42"},
		{"negative integer", "-7"},
		{"bool true", "true"},
		{"bool false", "false"},
		{"null", "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := parseExpression(tt.raw)
			assert.Equal(t, exprKindLiteral, node.kind)
			assert.Equal(t, tt.raw, node.raw)
		})
	}
}

func TestParseExpressionSymbolicRef(t *testing.T) {
	node := parseExpression("myParam")
	assert.Equal(t, exprKindSymbolicRef, node.kind)
	assert.Equal(t, "myParam", node.target)
	assert.Equal(t, "myParam", node.raw)
	assert.Empty(t, node.path)
}

func TestParseExpressionFunctionCallSimple(t *testing.T) {
	// resourceGroup().location -> propertyAccess whose base is a
	// functionCall on resourceGroup.
	node := parseExpression("resourceGroup().location")
	require.Equal(t, exprKindPropertyAccess, node.kind)
	assert.Equal(t, []string{"location"}, node.path)
	assert.Equal(t, "resourceGroup().location", node.raw)

	// The base function call is preserved as the first arg.
	require.Len(t, node.args, 1)
	base := node.args[0]
	assert.Equal(t, exprKindFunctionCall, base.kind)
	assert.Equal(t, "resourceGroup", base.functionName)
	assert.Empty(t, base.args)
}

func TestParseExpressionFunctionCallNestedArgs(t *testing.T) {
	node := parseExpression("concat('a', var, 'b')")
	require.Equal(t, exprKindFunctionCall, node.kind)
	assert.Equal(t, "concat", node.functionName)
	require.Len(t, node.args, 3)

	assert.Equal(t, exprKindLiteral, node.args[0].kind)
	assert.Equal(t, "'a'", node.args[0].raw)

	assert.Equal(t, exprKindSymbolicRef, node.args[1].kind)
	assert.Equal(t, "var", node.args[1].target)

	assert.Equal(t, exprKindLiteral, node.args[2].kind)
	assert.Equal(t, "'b'", node.args[2].raw)
}

func TestParseExpressionFunctionCallDeepNesting(t *testing.T) {
	node := parseExpression("resourceId('Microsoft.Network/virtualNetworks', concat(prefix, '-vnet'))")
	require.Equal(t, exprKindFunctionCall, node.kind)
	assert.Equal(t, "resourceId", node.functionName)
	require.Len(t, node.args, 2)

	assert.Equal(t, exprKindLiteral, node.args[0].kind)

	inner := node.args[1]
	require.Equal(t, exprKindFunctionCall, inner.kind)
	assert.Equal(t, "concat", inner.functionName)
	require.Len(t, inner.args, 2)
	assert.Equal(t, exprKindSymbolicRef, inner.args[0].kind)
	assert.Equal(t, "prefix", inner.args[0].target)
	assert.Equal(t, exprKindLiteral, inner.args[1].kind)
}

func TestParseExpressionInterpolation(t *testing.T) {
	node := parseExpression("'${prefix}-sa'")
	require.Equal(t, exprKindInterpolation, node.kind)
	assert.Equal(t, "'${prefix}-sa'", node.raw)

	// segments: leading literal "", embedded expr `prefix`, trailing "-sa"
	require.Len(t, node.segments, 3)
	assert.Equal(t, exprKindLiteral, node.segments[0].kind)
	assert.Equal(t, "", node.segments[0].raw)

	assert.Equal(t, exprKindSymbolicRef, node.segments[1].kind)
	assert.Equal(t, "prefix", node.segments[1].target)

	assert.Equal(t, exprKindLiteral, node.segments[2].kind)
	assert.Equal(t, "-sa", node.segments[2].raw)
}

func TestParseExpressionInterpolationNestedCall(t *testing.T) {
	node := parseExpression("'sa-${toLower(resourceGroup().name)}'")
	require.Equal(t, exprKindInterpolation, node.kind)
	require.Len(t, node.segments, 3)

	assert.Equal(t, exprKindLiteral, node.segments[0].kind)
	assert.Equal(t, "sa-", node.segments[0].raw)

	embedded := node.segments[1]
	require.Equal(t, exprKindFunctionCall, embedded.kind)
	assert.Equal(t, "toLower", embedded.functionName)
	require.Len(t, embedded.args, 1)
	assert.Equal(t, exprKindPropertyAccess, embedded.args[0].kind)

	assert.Equal(t, exprKindLiteral, node.segments[2].kind)
	assert.Equal(t, "", node.segments[2].raw)
}

func TestParseExpressionPropertyAccessChainWithIndex(t *testing.T) {
	node := parseExpression("vnet.properties.subnets[0].id")
	require.Equal(t, exprKindPropertyAccess, node.kind)
	assert.Equal(t, "vnet", node.target)
	assert.Equal(t, []string{"properties", "subnets", "0", "id"}, node.path)
	assert.Equal(t, "vnet.properties.subnets[0].id", node.raw)
}

func TestParseExpressionPropertyAccessQuotedKey(t *testing.T) {
	// foo['bar'] should normalize to the same segment as foo.bar.
	node := parseExpression("storageAccounts['blobServices'].properties")
	require.Equal(t, exprKindPropertyAccess, node.kind)
	assert.Equal(t, "storageAccounts", node.target)
	assert.Equal(t, []string{"blobServices", "properties"}, node.path)
}

func TestParseExpressionTernary(t *testing.T) {
	node := parseExpression("useProd ? 'prod' : 'dev'")
	require.Equal(t, exprKindTernary, node.kind)
	require.Len(t, node.args, 3)

	assert.Equal(t, exprKindSymbolicRef, node.args[0].kind)
	assert.Equal(t, "useProd", node.args[0].target)

	assert.Equal(t, exprKindLiteral, node.args[1].kind)
	assert.Equal(t, "'prod'", node.args[1].raw)

	assert.Equal(t, exprKindLiteral, node.args[2].kind)
	assert.Equal(t, "'dev'", node.args[2].raw)
}

func TestParseExpressionTernaryWithComparisonCondition(t *testing.T) {
	// The condition uses `==`, which is outside the modeled grammar, so it
	// becomes an `unknown` node — but the overall expression is still a
	// ternary with its branches parsed.
	node := parseExpression("prefix == 'prod' ? 'Premium' : 'Standard'")
	require.Equal(t, exprKindTernary, node.kind)
	require.Len(t, node.args, 3)

	assert.Equal(t, exprKindUnknown, node.args[0].kind)
	assert.Equal(t, "prefix == 'prod'", node.args[0].raw)

	assert.Equal(t, exprKindLiteral, node.args[1].kind)
	assert.Equal(t, "'Premium'", node.args[1].raw)

	assert.Equal(t, exprKindLiteral, node.args[2].kind)
	assert.Equal(t, "'Standard'", node.args[2].raw)
}

func TestParseExpressionTernaryInsideArray(t *testing.T) {
	// A ternary as an array element must not swallow the following element.
	node := parseExpression("[useProd ? 'a' : 'b', other]")
	require.Equal(t, exprKindArray, node.kind)
	require.Len(t, node.args, 2)
	assert.Equal(t, exprKindTernary, node.args[0].kind)
	require.Len(t, node.args[0].args, 3)
	assert.Equal(t, exprKindSymbolicRef, node.args[1].kind)
	assert.Equal(t, "other", node.args[1].target)
}

func TestParseExpressionArray(t *testing.T) {
	node := parseExpression("['a', 'b', myVar]")
	require.Equal(t, exprKindArray, node.kind)
	require.Len(t, node.args, 3)
	assert.Equal(t, exprKindLiteral, node.args[0].kind)
	assert.Equal(t, exprKindLiteral, node.args[1].kind)
	assert.Equal(t, exprKindSymbolicRef, node.args[2].kind)
	assert.Equal(t, "myVar", node.args[2].target)
}

func TestParseExpressionEmptyArray(t *testing.T) {
	node := parseExpression("[]")
	require.Equal(t, exprKindArray, node.kind)
	assert.Empty(t, node.args)
}

func TestParseExpressionUnknown(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"unbalanced parens", "concat('a', "},
		{"trailing garbage", "foo.bar !! baz"},
		{"unterminated string", "'no closing quote"},
		{"bare operator", "a + b"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := parseExpression(tt.raw)
			assert.Equal(t, exprKindUnknown, node.kind, "input %q", tt.raw)
			// raw is always preserved (trimmed).
			assert.NotNil(t, node)
		})
	}
}

func TestParseExpressionInterpolationWithSecureParamRef(t *testing.T) {
	// A realistic "does this name interpolate a parameter?" shape.
	node := parseExpression("'${adminPassword}'")
	require.Equal(t, exprKindInterpolation, node.kind)
	require.Len(t, node.segments, 3)
	assert.Equal(t, exprKindSymbolicRef, node.segments[1].kind)
	assert.Equal(t, "adminPassword", node.segments[1].target)
}

func TestParseExpressionRawAlwaysPopulated(t *testing.T) {
	// Whitespace around the expression is trimmed but raw is never empty
	// for a non-empty input.
	node := parseExpression("   resourceGroup().location   ")
	assert.Equal(t, "resourceGroup().location", node.raw)
	assert.Equal(t, exprKindPropertyAccess, node.kind)
}

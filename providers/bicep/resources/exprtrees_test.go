// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceModuleExpressionTrees drives the resource/module expression-tree
// accessors off a committed fixture, parsing each security-relevant raw field
// (name, location, condition, scope) through the same tokenizer the
// nameTree/locationTree/conditionTree/scopeTree accessors use, and asserts the
// produced node shape.
func TestResourceModuleExpressionTrees(t *testing.T) {
	data, err := os.ReadFile("testdata/exprtrees.bicep")
	require.NoError(t, err)

	result := parseBicep(string(data))

	resources := map[string]parsedResource{}
	for _, r := range result.resources {
		resources[r.symbolicName] = r
	}
	require.Contains(t, resources, "interpName")
	require.Contains(t, resources, "fnLoc")

	modules := map[string]parsedModule{}
	for _, m := range result.modules {
		modules[m.name] = m
	}
	require.Contains(t, modules, "net")

	t.Run("interpolated name", func(t *testing.T) {
		r := resources["interpName"]
		node := parseExpression(r.name)
		require.Equal(t, exprKindInterpolation, node.kind)
		assert.Equal(t, "'${prefix}-sa'", node.raw)
		require.Len(t, node.segments, 3)
		assert.Equal(t, exprKindSymbolicRef, node.segments[1].kind)
		assert.Equal(t, "prefix", node.segments[1].target)
		assert.Equal(t, "-sa", node.segments[2].raw)
	})

	t.Run("literal location", func(t *testing.T) {
		r := resources["interpName"]
		node := parseExpression(r.location)
		require.Equal(t, exprKindLiteral, node.kind)
		assert.Equal(t, "'eastus'", node.raw)
	})

	t.Run("unconditional resource yields unknown condition tree", func(t *testing.T) {
		r := resources["interpName"]
		assert.Equal(t, "", r.condition)
		node := parseExpression(r.condition)
		require.Equal(t, exprKindUnknown, node.kind)
		assert.Equal(t, "", node.raw)
	})

	t.Run("function-call location", func(t *testing.T) {
		r := resources["fnLoc"]
		node := parseExpression(r.location)
		// resourceGroup().location parses to a propertyAccess whose base is
		// a functionCall on resourceGroup.
		require.Equal(t, exprKindPropertyAccess, node.kind)
		assert.Equal(t, []string{"location"}, node.path)
		require.Len(t, node.args, 1)
		base := node.args[0]
		assert.Equal(t, exprKindFunctionCall, base.kind)
		assert.Equal(t, "resourceGroup", base.functionName)
	})

	t.Run("conditional resource condition tree", func(t *testing.T) {
		r := resources["fnLoc"]
		assert.Equal(t, "deployFlag", r.condition)
		node := parseExpression(r.condition)
		require.Equal(t, exprKindSymbolicRef, node.kind)
		assert.Equal(t, "deployFlag", node.target)
	})

	t.Run("module scope tree", func(t *testing.T) {
		m := modules["net"]
		node := parseExpression(m.scope)
		require.Equal(t, exprKindFunctionCall, node.kind)
		assert.Equal(t, "resourceGroup", node.functionName)
		require.Len(t, node.args, 1)
		assert.Equal(t, exprKindLiteral, node.args[0].kind)
		assert.Equal(t, "'rg-network'", node.args[0].raw)
	})

	t.Run("module condition tree", func(t *testing.T) {
		m := modules["net"]
		assert.Equal(t, "deployFlag", m.condition)
		node := parseExpression(m.condition)
		require.Equal(t, exprKindSymbolicRef, node.kind)
		assert.Equal(t, "deployFlag", node.target)
	})
}

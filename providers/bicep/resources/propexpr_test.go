// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectPropertyExpressions runs an accessor's []any of
// *mqlBicepPropertyExpression into a path->resource map for easy assertion.
func collectPropertyExpressions(t *testing.T, list []any) map[string]*mqlBicepPropertyExpression {
	t.Helper()
	out := map[string]*mqlBicepPropertyExpression{}
	for _, item := range list {
		pe, ok := item.(*mqlBicepPropertyExpression)
		require.True(t, ok)
		out[pe.Path.Data] = pe
	}
	return out
}

func TestPropertyExpressions(t *testing.T) {
	data, err := os.ReadFile("testdata/propexpr.bicep")
	require.NoError(t, err)

	parsed := parseBicep(string(data))
	resolver := newSymbolResolver("testdata/propexpr.bicep", parsed)
	runtime := testRuntime()

	saResource := findResource(parsed, "sa")
	require.Equal(t, "sa", saResource.symbolicName)

	sa, err := newMqlBicepResource(runtime, "bicep.resource:testdata/propexpr.bicep:sa", saResource, resolver, nil)
	require.NoError(t, err)

	list, err := sa.propertyExpressions()
	require.NoError(t, err)
	byPath := collectPropertyExpressions(t, list)

	t.Run("walks nested object into dotted paths", func(t *testing.T) {
		// Stable, sorted set of every string-valued leaf in properties.
		var paths []string
		for p := range byPath {
			paths = append(paths, p)
		}
		assert.ElementsMatch(t, []string{
			"accessTier",
			"adminCredentials.password",
			"derivedLocation",
			"encryption.keyVaultProperties.keyVaultUri",
		}, paths)
	})

	t.Run("property calling a built-in function is a functionCall", func(t *testing.T) {
		pe := byPath["derivedLocation"]
		require.NotNil(t, pe)
		expr, err := pe.expression()
		require.NoError(t, err)
		// resourceGroup().location -> propertyAccess whose base is a functionCall.
		require.Equal(t, exprKindPropertyAccess, expr.node.kind)
		require.Len(t, expr.node.args, 1)
		assert.Equal(t, exprKindFunctionCall, expr.node.args[0].kind)
		assert.Equal(t, "resourceGroup", expr.node.args[0].functionName)
		kind, err := expr.referenceKind()
		require.NoError(t, err)
		assert.Equal(t, "", kind)
	})

	t.Run("deep @secure parameter interpolation resolves to parameter", func(t *testing.T) {
		pe := byPath["adminCredentials.password"]
		require.NotNil(t, pe)
		expr, err := pe.expression()
		require.NoError(t, err)
		require.Equal(t, exprKindInterpolation, expr.node.kind)
		// The interpolation's embedded segment is the @secure parameter.
		seg, err := newMqlBicepExpression(runtime, "seg", expr.node.segments[1], resolver)
		require.NoError(t, err)
		kind, err := seg.referenceKind()
		require.NoError(t, err)
		assert.Equal(t, refKindParameter, kind)
		p, err := seg.referencedParameter()
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "adminPassword", p.Name.Data)
		assert.True(t, p.Secure.Data)
	})

	t.Run("property referencing another resource resolves to resource", func(t *testing.T) {
		pe := byPath["encryption.keyVaultProperties.keyVaultUri"]
		require.NotNil(t, pe)
		expr, err := pe.expression()
		require.NoError(t, err)
		// kv.properties.vaultUri — propertyAccess rooted at the kv resource.
		require.Equal(t, exprKindPropertyAccess, expr.node.kind)
		assert.Equal(t, "kv", expr.node.target)
		kind, err := expr.referenceKind()
		require.NoError(t, err)
		assert.Equal(t, refKindResource, kind)
		res, err := expr.referencedResource()
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "kv", res.SymbolicName.Data)
	})

	t.Run("hardcoded literal property is a literal", func(t *testing.T) {
		pe := byPath["accessTier"]
		require.NotNil(t, pe)
		expr, err := pe.expression()
		require.NoError(t, err)
		assert.Equal(t, exprKindLiteral, expr.node.kind)
		kind, err := expr.referenceKind()
		require.NoError(t, err)
		assert.Equal(t, "", kind)
	})
}

func TestParamExpressions(t *testing.T) {
	data, err := os.ReadFile("testdata/propexpr.bicep")
	require.NoError(t, err)

	parsed := parseBicep(string(data))
	resolver := newSymbolResolver("testdata/propexpr.bicep", parsed)
	runtime := testRuntime()

	var netModule parsedModule
	for _, m := range parsed.modules {
		if m.name == "net" {
			netModule = m
		}
	}
	require.Equal(t, "net", netModule.name)

	mods, err := createMqlModules(runtime, "testdata/propexpr.bicep", []parsedModule{netModule}, resolver)
	require.NoError(t, err)
	net := mods[0].(*mqlBicepModule)

	list, err := net.paramExpressions()
	require.NoError(t, err)
	byPath := collectPropertyExpressions(t, list)

	t.Run("walks params into entries", func(t *testing.T) {
		var paths []string
		for p := range byPath {
			paths = append(paths, p)
		}
		assert.ElementsMatch(t, []string{"region", "secret"}, paths)
	})

	t.Run("param value forwarding a parameter resolves to parameter", func(t *testing.T) {
		pe := byPath["secret"]
		require.NotNil(t, pe)
		expr, err := pe.expression()
		require.NoError(t, err)
		require.Equal(t, exprKindSymbolicRef, expr.node.kind)
		kind, err := expr.referenceKind()
		require.NoError(t, err)
		assert.Equal(t, refKindParameter, kind)
		p, err := expr.referencedParameter()
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "adminPassword", p.Name.Data)
	})
}

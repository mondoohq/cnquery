// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTypeDecl(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		decorators  []string
		wantName    string
		wantDef     string
		wantExport  bool
		wantDesc    string
		wantSkipped bool
	}{
		{
			name:     "string-literal union",
			input:    "type sku = 'Standard_LRS' | 'Premium_LRS'",
			wantName: "sku",
			wantDef:  "'Standard_LRS' | 'Premium_LRS'",
		},
		{
			name: "multi-line object type",
			input: `type cfg = {
  name: string
  tier: sku
}`,
			wantName: "cfg",
			wantDef:  "{ name: string tier: sku }",
		},
		{
			name:       "exported with description",
			input:      "type shared = string",
			decorators: []string{"@description('a shared type')", "@export()"},
			wantName:   "shared",
			wantDef:    "string",
			wantExport: true,
			wantDesc:   "a shared type",
		},
		{
			name:        "not a type decl",
			input:       "typeof x",
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseTypeDecl(tt.input, tt.decorators)
			if tt.wantSkipped {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tt.wantName, got.name)
			assert.Equal(t, tt.wantDef, got.definition)
			assert.Equal(t, tt.wantExport, got.exported)
			assert.Equal(t, tt.wantDesc, got.description)
		})
	}
}

func TestParseFunctionDecl(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		decorators  []string
		wantName    string
		wantParams  map[string]string
		wantReturn  string
		wantExpr    string
		wantDesc    string
		wantSkipped bool
	}{
		{
			name:       "two params",
			input:      "func buildName(prefix string, idx int) string => '${prefix}-${idx}'",
			wantName:   "buildName",
			wantParams: map[string]string{"prefix": "string", "idx": "int"},
			wantReturn: "string",
			wantExpr:   "'${prefix}-${idx}'",
		},
		{
			name:       "no params with description",
			input:      "func answer() int => 42",
			decorators: []string{"@description('the answer')"},
			wantName:   "answer",
			wantParams: nil,
			wantReturn: "int",
			wantExpr:   "42",
			wantDesc:   "the answer",
		},
		{
			// Object-typed parameter contains `{`, `}`, and a `,` that must
			// not split the parameter list, plus a nested type.
			name:       "object-typed parameter",
			input:      "func render(opts { name: string, tier: int }) string => opts.name",
			wantName:   "render",
			wantParams: map[string]string{"opts": "{ name: string, tier: int }"},
			wantReturn: "string",
			wantExpr:   "opts.name",
		},
		{
			// Array return type (`string[]`) and a lambda `=>` inside the body
			// that must not be mistaken for the function arrow.
			name:       "array return type with lambda body",
			input:      "func names(items array) string[] => map(items, i => i.name)",
			wantName:   "names",
			wantParams: map[string]string{"items": "array"},
			wantReturn: "string[]",
			wantExpr:   "map(items, i => i.name)",
		},
		{
			name:        "not a func decl",
			input:       "function() {}",
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFunctionDecl(tt.input, tt.decorators)
			if tt.wantSkipped {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tt.wantName, got.name)
			assert.Equal(t, tt.wantParams, got.parameters)
			assert.Equal(t, tt.wantReturn, got.returnType)
			assert.Equal(t, tt.wantExpr, got.expression)
			assert.Equal(t, tt.wantDesc, got.description)
		})
	}
}

func TestParseImportDecl(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantSource    string
		wantSymbols   []string
		wantNamespace string
		wantWildcard  bool
		wantSkipped   bool
	}{
		{
			name:        "named imports",
			input:       "import { typeA, funcB } from './shared.bicep'",
			wantSource:  "./shared.bicep",
			wantSymbols: []string{"typeA", "funcB"},
		},
		{
			name:          "wildcard namespace import",
			input:         "import * as shared from './shared.bicep'",
			wantSource:    "./shared.bicep",
			wantNamespace: "shared",
			wantWildcard:  true,
		},
		{
			name:       "bare provider import",
			input:      "import 'az@2.0.0'",
			wantSource: "az@2.0.0",
		},
		{
			name:        "not an import",
			input:       "important()",
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseImportDecl(tt.input)
			if tt.wantSkipped {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tt.wantSource, got.source)
			assert.Equal(t, tt.wantSymbols, got.symbols)
			assert.Equal(t, tt.wantNamespace, got.namespace)
			assert.Equal(t, tt.wantWildcard, got.wildcard)
		})
	}
}

func TestParseMetadataDecl(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "literal string",
			input:     "metadata description = 'My template'",
			wantName:  "description",
			wantValue: "My template",
			wantOk:    true,
		},
		{
			name:   "expression-valued metadata is skipped",
			input:  "metadata config = { tier: 'gold' }",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, value, ok := parseMetadataDecl(tt.input)
			assert.Equal(t, tt.wantOk, ok)
			if !tt.wantOk {
				return
			}
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestParseBicepDeclarationsFixture(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "declarations.bicep"))
	require.NoError(t, err)

	result := parseBicep(string(content))

	// types
	require.Len(t, result.types, 2)
	sku := result.types[0]
	assert.Equal(t, "sku", sku.name)
	assert.Equal(t, "'Standard_LRS' | 'Premium_LRS'", sku.definition)
	assert.True(t, sku.exported)
	assert.Equal(t, "Storage SKU", sku.description)
	cfg := result.types[1]
	assert.Equal(t, "cfg", cfg.name)
	assert.Equal(t, "{ name: string tier: sku }", cfg.definition)
	assert.False(t, cfg.exported)

	// functions
	require.Len(t, result.functions, 1)
	fn := result.functions[0]
	assert.Equal(t, "buildName", fn.name)
	assert.Equal(t, map[string]string{"prefix": "string", "idx": "int"}, fn.parameters)
	assert.Equal(t, "string", fn.returnType)
	assert.Equal(t, "'${prefix}-${idx}'", fn.expression)
	assert.Equal(t, "Build a resource name", fn.description)

	// imports — named, wildcard, provider
	require.Len(t, result.imports, 3)
	assert.Equal(t, "./shared.bicep", result.imports[0].source)
	assert.Equal(t, []string{"typeA", "funcB"}, result.imports[0].symbols)
	assert.False(t, result.imports[0].wildcard)

	assert.Equal(t, "./shared.bicep", result.imports[1].source)
	assert.Equal(t, "shared", result.imports[1].namespace)
	assert.True(t, result.imports[1].wildcard)
	assert.Empty(t, result.imports[1].symbols)

	assert.Equal(t, "az@2.0.0", result.imports[2].source)
	assert.Empty(t, result.imports[2].symbols)
	assert.False(t, result.imports[2].wildcard)

	// metadata
	assert.Equal(t, map[string]string{
		"description": "My template",
		"author":      "platform-team",
	}, result.metadata)

	// existing constructs still parse
	require.Len(t, result.parameters, 1)
	assert.Equal(t, "location", result.parameters[0].name)
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeBicep(t *testing.T) {
	tests := []struct {
		name string
		// want describes each expected statement in order.
		want []struct {
			keyword    string
			decorators []string
			// containsAll asserts substrings that must appear in the
			// statement's reassembled text.
			containsAll []string
		}
		input string
	}{
		{
			name:  "simple single-line param",
			input: `param location string = 'eastus'`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "param", containsAll: []string{"param location string = 'eastus'"}},
			},
		},
		{
			name: "multi-line param with object default",
			input: `param config object = {
  name: 'myService'
  enabled: true
}`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "param", containsAll: []string{"param config object = {", "name: 'myService'", "enabled: true", "}"}},
			},
		},
		{
			name: "resource with nested body containing brace and comment in a string",
			input: `resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorage'
  properties: {
    note: 'closing brace } and // not a comment'
  }
}`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "resource", containsAll: []string{
					"resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {",
					"note: 'closing brace } and // not a comment'",
					"properties: {",
				}},
			},
		},
		{
			name: "var with multi-line array",
			input: `var regions = [
  'eastus'
  'westus'
]`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "var", containsAll: []string{"var regions = [", "'eastus'", "'westus'", "]"}},
			},
		},
		{
			name: "stacked multi-line decorators attach to following statement",
			input: `@allowed([
  'Standard_LRS'
  'Standard_GRS'
])
@secure()
param storageSku string`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{
					keyword: "param",
					decorators: []string{
						"@allowed([\n'Standard_LRS'\n'Standard_GRS'\n])",
						"@secure()",
					},
					containsAll: []string{"param storageSku string"},
				},
			},
		},
		{
			name: "blank and comment lines between statements are skipped",
			input: `param a string

// a leading comment
var b = 'x'

// trailing comment block
output c string = b`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "param", containsAll: []string{"param a string"}},
				{keyword: "var", containsAll: []string{"var b = 'x'"}},
				{keyword: "output", containsAll: []string{"output c string = b"}},
			},
		},
		{
			name:  "unknown leading keyword becomes empty-keyword statement",
			input: `unknownKeyword myThing = string`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "", containsAll: []string{"unknownKeyword myThing = string"}},
			},
		},
		{
			name: "type, func, import, and metadata are classified",
			input: `type sku = 'a' | 'b'
func f() int => 1
import 'az@2.0.0'
metadata author = 'team'`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "type", containsAll: []string{"type sku = 'a' | 'b'"}},
				{keyword: "func", containsAll: []string{"func f() int => 1"}},
				{keyword: "import", containsAll: []string{"import 'az@2.0.0'"}},
				{keyword: "metadata", containsAll: []string{"metadata author = 'team'"}},
			},
		},
		{
			name:  "targetScope is classified",
			input: `targetScope = 'subscription'`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "targetScope", containsAll: []string{"targetScope = 'subscription'"}},
			},
		},
		{
			name: "mixed file yields all statements in order",
			input: `targetScope = 'subscription'

@description('region')
param location string = 'eastus'

var rgName = 'myRG'

resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: rgName
}

module net './net.bicep' = {
  name: 'netDeploy'
}

output rgId string = rg.id`,
			want: []struct {
				keyword     string
				decorators  []string
				containsAll []string
			}{
				{keyword: "targetScope"},
				{keyword: "param", decorators: []string{"@description('region')"}},
				{keyword: "var"},
				{keyword: "resource"},
				{keyword: "module"},
				{keyword: "output"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts := tokenizeBicep(tt.input)
			require.Len(t, stmts, len(tt.want), "statement count")
			for i, w := range tt.want {
				assert.Equal(t, w.keyword, stmts[i].keyword, "statement %d keyword", i)
				if w.decorators != nil {
					assert.Equal(t, w.decorators, stmts[i].decorators, "statement %d decorators", i)
				}
				for _, sub := range w.containsAll {
					assert.Contains(t, stmts[i].text, sub, "statement %d text", i)
				}
				assert.Positive(t, stmts[i].startLine, "statement %d startLine should be 1-based", i)
			}
		})
	}
}

// A resource body whose string literal contains an unbalanced `}` must not
// terminate the statement early — the tokenizer relies on scanState to ignore
// delimiters inside strings.
func TestTokenizeBicep_StringDelimitersDoNotCloseEarly(t *testing.T) {
	input := `resource sa 'Type@ver' = {
  note: '} ] )'
  name: 'real'
}

output done string = sa.name`
	stmts := tokenizeBicep(input)
	require.Len(t, stmts, 2)
	assert.Equal(t, "resource", stmts[0].keyword)
	assert.Contains(t, stmts[0].text, "name: 'real'")
	assert.Equal(t, "output", stmts[1].keyword)
}

// Decorators that run to end-of-file with no following statement should still
// surface as an empty-keyword statement so nothing is silently dropped.
func TestTokenizeBicep_TrailingDecoratorOnly(t *testing.T) {
	input := `@description('dangling')`
	stmts := tokenizeBicep(input)
	require.Len(t, stmts, 1)
	assert.Equal(t, "", stmts[0].keyword)
	assert.Equal(t, []string{"@description('dangling')"}, stmts[0].decorators)
}

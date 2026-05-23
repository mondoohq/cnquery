// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBicepFullFile(t *testing.T) {
	input := `targetScope = 'subscription'

@description('The Azure region')
param location string = 'eastus'

var rgName = 'myResourceGroup'

resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: rgName
  location: location
}

module network './modules/network.bicep' = {
  name: 'networkDeploy'
  scope: rg
  params: {
    location: location
  }
}

output rgId string = rg.id
`

	result := parseBicep(input)

	assert.Equal(t, "subscription", result.targetScope)
	assert.Len(t, result.parameters, 1)
	assert.Len(t, result.variables, 1)
	assert.Len(t, result.resources, 1)
	assert.Len(t, result.modules, 1)
	assert.Len(t, result.outputs, 1)
}

func TestParseBicepNoTargetScope(t *testing.T) {
	input := `param name string`
	result := parseBicep(input)
	assert.Equal(t, "", result.targetScope)
	assert.Len(t, result.parameters, 1)
}

func TestParseParameterDefaultValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantType string
		wantDef  string
	}{
		{
			name:     "empty string default",
			input:    "param teradiciRegKey string = ''",
			wantName: "teradiciRegKey",
			wantType: "string",
			wantDef:  "",
		},
		{
			name:     "non-empty string default",
			input:    "param location string = 'eastus'",
			wantName: "location",
			wantType: "string",
			wantDef:  "eastus",
		},
		{
			name:     "int default not quoted",
			input:    "param count int = 3",
			wantName: "count",
			wantType: "int",
			wantDef:  "3",
		},
		{
			name:     "bool default not quoted",
			input:    "param enabled bool = true",
			wantName: "enabled",
			wantType: "bool",
			wantDef:  "true",
		},
		{
			name:     "no default value",
			input:    "param name string",
			wantName: "name",
			wantType: "string",
			wantDef:  "",
		},
		{
			name:     "string with content",
			input:    "param sku string = 'Standard_LRS'",
			wantName: "sku",
			wantType: "string",
			wantDef:  "Standard_LRS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parseParameter(tt.input, nil)
			assert.Equal(t, tt.wantName, p.name)
			assert.Equal(t, tt.wantType, p.typ)
			assert.Equal(t, tt.wantDef, p.defaultValue)
		})
	}
}

func TestParseParameterDecorators(t *testing.T) {
	input := `@description('Teradici Registration Key')
@secure()
param teradiciRegKey string = ''`

	result := parseBicep(input)
	require.Len(t, result.parameters, 1)
	p := result.parameters[0]

	assert.Equal(t, "teradiciRegKey", p.name)
	assert.Equal(t, "string", p.typ)
	assert.Equal(t, "", p.defaultValue)
	assert.Equal(t, "Teradici Registration Key", p.description)
	assert.True(t, p.secure)
	assert.Equal(t, []string{
		"@description('Teradici Registration Key')",
		"@secure()",
	}, p.decorators)
}

func TestParseParameterAllowed(t *testing.T) {
	t.Run("multiline newline-separated", func(t *testing.T) {
		input := `@allowed([
  'Standard_LRS'
  'Standard_GRS'
])
param storageSku string`

		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		p := result.parameters[0]

		assert.Equal(t, "storageSku", p.name)
		assert.Equal(t, []string{"Standard_LRS", "Standard_GRS"}, p.allowed)
	})

	t.Run("single-line comma-separated", func(t *testing.T) {
		input := `@allowed([ 'win10', 'ws2019' ])
param osType string = 'win10'`

		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		p := result.parameters[0]

		assert.Equal(t, "osType", p.name)
		assert.Equal(t, []string{"win10", "ws2019"}, p.allowed)
	})

	t.Run("mixed comma and newline", func(t *testing.T) {
		input := `@allowed([
  'a', 'b'
  'c'
])
param x string`

		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		assert.Equal(t, []string{"a", "b", "c"}, result.parameters[0].allowed)
	})
}

func TestParseAllowedValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "newline-separated",
			raw:  "'a'\n'b'\n'c'",
			want: []string{"a", "b", "c"},
		},
		{
			name: "comma-separated on one line",
			raw:  " 'win10', 'ws2019' ",
			want: []string{"win10", "ws2019"},
		},
		{
			name: "empty input",
			raw:  "",
			want: nil,
		},
		{
			name: "whitespace only",
			raw:  "   \n   ",
			want: nil,
		},
		{
			name: "empty string value",
			raw:  "''",
			want: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAllowedValues(tt.raw)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseVariable(t *testing.T) {
	t.Run("simple expression", func(t *testing.T) {
		v := parseVariable("var rgName = 'myRG'", nil)
		assert.Equal(t, "rgName", v.name)
		assert.Equal(t, "'myRG'", v.expression)
	})

	t.Run("complex expression", func(t *testing.T) {
		v := parseVariable("var subnetRef = resourceId('Microsoft.Network/virtualNetworks/subnets', vnetName, subnetName)", nil)
		assert.Equal(t, "subnetRef", v.name)
		assert.Equal(t, "resourceId('Microsoft.Network/virtualNetworks/subnets', vnetName, subnetName)", v.expression)
	})

	t.Run("with description decorator", func(t *testing.T) {
		input := `@description('Resource group name')
var rgName = 'myRG'`
		result := parseBicep(input)
		require.Len(t, result.variables, 1)
		assert.Equal(t, "rgName", result.variables[0].name)
		assert.Equal(t, "Resource group name", result.variables[0].description)
	})
}

func TestParseResourceDecl(t *testing.T) {
	t.Run("basic resource", func(t *testing.T) {
		input := `resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: storageName
  location: location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
}`
		lines := splitLines(input)
		res, consumed := parseResourceDecl(lines, 0, nil)
		require.NotNil(t, res)
		assert.Equal(t, "storageAccount", res.symbolicName)
		assert.Equal(t, "Microsoft.Storage/storageAccounts", res.typ)
		assert.Equal(t, "2023-01-01", res.apiVersion)
		assert.Equal(t, "storageName", res.name)
		assert.Equal(t, "location", res.location)
		assert.False(t, res.existing)
		assert.Equal(t, len(lines), consumed)
	})

	t.Run("existing resource reference", func(t *testing.T) {
		input := `resource vnet 'Microsoft.Network/virtualNetworks@2023-04-01' existing = {
  name: 'myVnet'
}`
		lines := splitLines(input)
		res, _ := parseResourceDecl(lines, 0, nil)
		require.NotNil(t, res)
		assert.True(t, res.existing)
		assert.Equal(t, "vnet", res.symbolicName)
	})

	t.Run("resource with condition", func(t *testing.T) {
		input := `resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = if (deployStorage) {
  name: storageName
  location: location
}`
		lines := splitLines(input)
		res, _ := parseResourceDecl(lines, 0, nil)
		require.NotNil(t, res)
		assert.Equal(t, "deployStorage", res.condition)
	})

	t.Run("resource with parent", func(t *testing.T) {
		input := `resource subnet 'Microsoft.Network/virtualNetworks/subnets@2023-04-01' = {
  name: 'mySubnet'
  parent: vnet
  properties: {
    addressPrefix: '10.0.0.0/24'
  }
}`
		lines := splitLines(input)
		res, _ := parseResourceDecl(lines, 0, nil)
		require.NotNil(t, res)
		assert.Equal(t, "vnet", res.parent)
	})

	t.Run("resource with dependsOn", func(t *testing.T) {
		input := `resource app 'Microsoft.Web/sites@2022-09-01' = {
  name: 'myApp'
  location: location
  dependsOn: [
    storageAccount
    appPlan
  ]
}`
		lines := splitLines(input)
		res, _ := parseResourceDecl(lines, 0, nil)
		require.NotNil(t, res)
		assert.Equal(t, []string{"storageAccount", "appPlan"}, res.dependsOn)
	})

	t.Run("resource with decorators", func(t *testing.T) {
		input := `resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'test'
}`
		decorators := []string{"@description('My storage account')"}
		lines := splitLines(input)
		res, _ := parseResourceDecl(lines, 0, decorators)
		require.NotNil(t, res)
		assert.Equal(t, decorators, res.decorators)
	})

	t.Run("type without api version", func(t *testing.T) {
		input := `resource rg 'Microsoft.Resources/resourceGroups' = {
  name: 'myRG'
}`
		lines := splitLines(input)
		res, _ := parseResourceDecl(lines, 0, nil)
		require.NotNil(t, res)
		assert.Equal(t, "Microsoft.Resources/resourceGroups", res.typ)
		assert.Equal(t, "", res.apiVersion)
	})
}

func TestParseModuleDecl(t *testing.T) {
	t.Run("local module", func(t *testing.T) {
		input := `module network './modules/network.bicep' = {
  name: 'networkDeploy'
  scope: rg
  params: {
    location: location
    vnetName: 'myVnet'
  }
}`
		lines := splitLines(input)
		mod, consumed := parseModuleDecl(lines, 0, nil)
		require.NotNil(t, mod)
		assert.Equal(t, "network", mod.name)
		assert.Equal(t, "./modules/network.bicep", mod.source)
		assert.Equal(t, "rg", mod.scope)
		assert.False(t, mod.isRegistry)
		assert.False(t, mod.isTemplateSpec)
		assert.Equal(t, len(lines), consumed)
	})

	t.Run("registry module br:", func(t *testing.T) {
		input := `module registry 'br:myregistry.azurecr.io/bicep/modules/storage:v1' = {
  name: 'registryDeploy'
}`
		lines := splitLines(input)
		mod, _ := parseModuleDecl(lines, 0, nil)
		require.NotNil(t, mod)
		assert.True(t, mod.isRegistry)
		assert.False(t, mod.isTemplateSpec)
	})

	t.Run("registry module br/", func(t *testing.T) {
		input := `module registry 'br/public:avm/res/storage/storage-account:0.9.0' = {
  name: 'registryDeploy'
}`
		lines := splitLines(input)
		mod, _ := parseModuleDecl(lines, 0, nil)
		require.NotNil(t, mod)
		assert.True(t, mod.isRegistry)
	})

	t.Run("template spec module ts:", func(t *testing.T) {
		input := `module ts 'ts:mySubscription/myRG/mySpec:v1' = {
  name: 'tsDeploy'
}`
		lines := splitLines(input)
		mod, _ := parseModuleDecl(lines, 0, nil)
		require.NotNil(t, mod)
		assert.True(t, mod.isTemplateSpec)
		assert.False(t, mod.isRegistry)
	})

	t.Run("template spec module ts/", func(t *testing.T) {
		input := `module ts 'ts/myAlias:mySpec:v1' = {
  name: 'tsDeploy'
}`
		lines := splitLines(input)
		mod, _ := parseModuleDecl(lines, 0, nil)
		require.NotNil(t, mod)
		assert.True(t, mod.isTemplateSpec)
	})

	t.Run("module with condition", func(t *testing.T) {
		input := `module optionalNet './modules/network.bicep' = if (deployNet) {
  name: 'netDeploy'
}`
		lines := splitLines(input)
		mod, _ := parseModuleDecl(lines, 0, nil)
		require.NotNil(t, mod)
		assert.Equal(t, "deployNet", mod.condition)
	})

	t.Run("module with decorators", func(t *testing.T) {
		input := `module network './modules/network.bicep' = {
  name: 'networkDeploy'
}`
		decorators := []string{"@description('Deploy the network module')"}
		lines := splitLines(input)
		mod, _ := parseModuleDecl(lines, 0, decorators)
		require.NotNil(t, mod)
		assert.Equal(t, "Deploy the network module", mod.description)
		assert.Equal(t, decorators, mod.decorators)
	})

	t.Run("non-matching line returns nil", func(t *testing.T) {
		lines := splitLines("var x = 1")
		mod, consumed := parseModuleDecl(lines, 0, nil)
		assert.Nil(t, mod)
		assert.Equal(t, 1, consumed)
	})
}

func TestParseOutput(t *testing.T) {
	t.Run("simple output", func(t *testing.T) {
		o := parseOutput("output rgId string = rg.id", nil)
		assert.Equal(t, "rgId", o.name)
		assert.Equal(t, "string", o.typ)
		assert.Equal(t, "rg.id", o.expression)
	})

	t.Run("output with description", func(t *testing.T) {
		input := `@description('The resource group ID')
output rgId string = rg.id`
		result := parseBicep(input)
		require.Len(t, result.outputs, 1)
		assert.Equal(t, "The resource group ID", result.outputs[0].description)
	})

	t.Run("int output", func(t *testing.T) {
		o := parseOutput("output count int = length(items)", nil)
		assert.Equal(t, "count", o.name)
		assert.Equal(t, "int", o.typ)
		assert.Equal(t, "length(items)", o.expression)
	})
}

func TestExtractBlock(t *testing.T) {
	t.Run("simple block", func(t *testing.T) {
		lines := splitLines("resource sa 'Type@ver' = {\n  name: 'test'\n}")
		block, consumed := extractBlock(lines, 0)
		assert.Contains(t, block, "name: 'test'")
		assert.Equal(t, 3, consumed)
	})

	t.Run("nested blocks", func(t *testing.T) {
		lines := splitLines("{\n  sku: {\n    name: 'Standard'\n  }\n}")
		block, consumed := extractBlock(lines, 0)
		assert.Contains(t, block, "sku: {")
		assert.Contains(t, block, "name: 'Standard'")
		assert.Equal(t, 5, consumed)
	})

	t.Run("no braces returns empty", func(t *testing.T) {
		lines := splitLines("no braces here")
		block, consumed := extractBlock(lines, 0)
		assert.Equal(t, "", block)
		assert.Equal(t, 1, consumed)
	})

	t.Run("unclosed block returns all remaining", func(t *testing.T) {
		lines := splitLines("{\n  name: 'test'")
		block, consumed := extractBlock(lines, 0)
		assert.Contains(t, block, "name: 'test'")
		assert.Equal(t, 2, consumed)
	})
}

func TestExtractFieldValue(t *testing.T) {
	body := `{
  name: storageName
  location: location
  kind: 'StorageV2'
}`

	assert.Equal(t, "storageName", extractFieldValue(body, "name"))
	assert.Equal(t, "location", extractFieldValue(body, "location"))
	assert.Equal(t, "", extractFieldValue(body, "parent"))
}

func TestExtractFieldValueUncachedField(t *testing.T) {
	body := "  custom: myValue"
	assert.Equal(t, "myValue", extractFieldValue(body, "custom"))
}

func TestExtractCondition(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "simple condition",
			line: "resource sa 'Type@ver' = if (deployStorage) {",
			want: "deployStorage",
		},
		{
			name: "nested parens",
			line: "resource sa 'Type@ver' = if (contains(env, 'prod')) {",
			want: "contains(env, 'prod')",
		},
		{
			name: "no condition",
			line: "resource sa 'Type@ver' = {",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractCondition(tt.line))
		})
	}
}

func TestExtractFieldBlock(t *testing.T) {
	t.Run("params block", func(t *testing.T) {
		body := `{
  name: 'deploy'
  params: {
    location: location
    sku: 'Standard'
  }
}`
		result := extractFieldBlock(body, "params")
		assert.Contains(t, result, "location: location")
		assert.Contains(t, result, "sku: 'Standard'")
	})

	t.Run("field not found", func(t *testing.T) {
		body := `{
  name: 'deploy'
}`
		result := extractFieldBlock(body, "params")
		assert.Equal(t, "", result)
	})

	t.Run("nested blocks within field", func(t *testing.T) {
		body := `{
  params: {
    config: {
      nested: true
    }
  }
}`
		result := extractFieldBlock(body, "params")
		assert.Contains(t, result, "config: {")
		assert.Contains(t, result, "nested: true")
	})
}

func TestExtractDependsOn(t *testing.T) {
	t.Run("multiple dependencies", func(t *testing.T) {
		body := `{
  name: 'test'
  dependsOn: [
    storageAccount
    appPlan
  ]
}`
		deps := extractDependsOn(body)
		assert.Equal(t, []string{"storageAccount", "appPlan"}, deps)
	})

	t.Run("single dependency", func(t *testing.T) {
		body := `{
  dependsOn: [
    vnet
  ]
}`
		deps := extractDependsOn(body)
		assert.Equal(t, []string{"vnet"}, deps)
	})

	t.Run("no dependsOn", func(t *testing.T) {
		body := `{
  name: 'test'
}`
		deps := extractDependsOn(body)
		assert.Nil(t, deps)
	})

	t.Run("inline dependencies", func(t *testing.T) {
		body := `{
  dependsOn: [storageAccount]
}`
		deps := extractDependsOn(body)
		assert.Equal(t, []string{"storageAccount"}, deps)
	})
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func TestParseParameterBounds(t *testing.T) {
	t.Run("string length bounds", func(t *testing.T) {
		input := `@minLength(8)
@maxLength(64)
param adminPassword string`
		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		p := result.parameters[0]
		assert.Equal(t, "adminPassword", p.name)
		require.NotNil(t, p.minLength)
		assert.Equal(t, int64(8), *p.minLength)
		require.NotNil(t, p.maxLength)
		assert.Equal(t, int64(64), *p.maxLength)
		assert.Nil(t, p.minValue)
		assert.Nil(t, p.maxValue)
	})

	t.Run("int value bounds", func(t *testing.T) {
		input := `@minValue(1)
@maxValue(1000)
param replicaCount int`
		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		p := result.parameters[0]
		require.NotNil(t, p.minValue)
		assert.Equal(t, int64(1), *p.minValue)
		require.NotNil(t, p.maxValue)
		assert.Equal(t, int64(1000), *p.maxValue)
		assert.Nil(t, p.minLength)
		assert.Nil(t, p.maxLength)
	})

	t.Run("no bounds decorators", func(t *testing.T) {
		input := `param simple string`
		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		p := result.parameters[0]
		assert.Nil(t, p.minLength)
		assert.Nil(t, p.maxLength)
		assert.Nil(t, p.minValue)
		assert.Nil(t, p.maxValue)
	})

	t.Run("explicit zero is distinct from absent", func(t *testing.T) {
		input := `@minValue(0)
@maxValue(0)
param zeroBounds int`
		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		p := result.parameters[0]
		require.NotNil(t, p.minValue)
		assert.Equal(t, int64(0), *p.minValue)
		require.NotNil(t, p.maxValue)
		assert.Equal(t, int64(0), *p.maxValue)
	})

	t.Run("malformed argument is ignored", func(t *testing.T) {
		input := `@minLength(abc)
param thing string`
		result := parseBicep(input)
		require.Len(t, result.parameters, 1)
		assert.Nil(t, result.parameters[0].minLength)
	})
}

func TestParseResourceTags(t *testing.T) {
	t.Run("literal string tags", func(t *testing.T) {
		input := `resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorage'
  location: 'eastus'
  tags: {
    env: 'prod'
    owner: 'platform-team'
    'cost-center': '1234'
  }
}`
		result := parseBicep(input)
		require.Len(t, result.resources, 1)
		assert.Equal(t, map[string]string{
			"env":         "prod",
			"owner":       "platform-team",
			"cost-center": "1234",
		}, result.resources[0].tags)
	})

	t.Run("expression-valued tags are skipped", func(t *testing.T) {
		input := `resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorage'
  tags: {
    env: parameters('env')
    owner: 'platform-team'
  }
}`
		result := parseBicep(input)
		require.Len(t, result.resources, 1)
		// Only the literal entry is captured; the expression-valued one is
		// dropped (audits can still reach it through `properties`).
		assert.Equal(t, map[string]string{"owner": "platform-team"}, result.resources[0].tags)
	})

	t.Run("no tags block", func(t *testing.T) {
		input := `resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorage'
}`
		result := parseBicep(input)
		require.Len(t, result.resources, 1)
		assert.Nil(t, result.resources[0].tags)
	})
}

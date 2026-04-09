// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestParseModuleDecorators(t *testing.T) {
	input := `@description('Deploy the network module')
module network './modules/network.bicep' = {
  name: 'networkDeploy'
  params: {
    location: location
  }
}`

	result := parseBicep(input)
	require.Len(t, result.modules, 1)
	m := result.modules[0]

	assert.Equal(t, "network", m.name)
	assert.Equal(t, "./modules/network.bicep", m.source)
	assert.Equal(t, "Deploy the network module", m.description)
	assert.Equal(t, []string{"@description('Deploy the network module')"}, m.decorators)
}

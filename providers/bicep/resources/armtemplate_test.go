// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/bicep/connection"
)

// armTemplateRuntime builds a runtime backed by a real BicepConnection
// scanning the armtemplate fixture directory, plus an in-memory resource
// cache so CreateResource dedupes by __id.
func armTemplateRuntime(t *testing.T) *plugin.Runtime {
	t.Helper()
	dir := filepath.Join("testdata", "armtemplate")
	asset := &inventory.Asset{
		Connections: []*inventory.Config{
			{Type: "bicep", Options: map[string]string{"path": dir}},
		},
	}
	conn, err := connection.NewBicepConnection(0, asset, asset.Connections[0])
	require.NoError(t, err)
	return &plugin.Runtime{
		Connection: conn,
		Resources:  &mapResources{m: map[string]plugin.Resource{}},
	}
}

func armTemplate(t *testing.T, runtime *plugin.Runtime) *mqlBicepTemplate {
	t.Helper()
	conn := runtime.Connection.(*connection.BicepConnection)
	mqlT, err := newMqlBicepTemplate(runtime, conn.Path(), conn.ARMTemplate())
	require.NoError(t, err)
	return mqlT
}

func TestARMTemplateParameters(t *testing.T) {
	runtime := armTemplateRuntime(t)
	tmpl := armTemplate(t, runtime)

	params, err := tmpl.parameters()
	require.NoError(t, err)
	require.Len(t, params, 2)

	// Stable, name-sorted order: adminPassword, instanceCount.
	admin := params[0].(*mqlBicepTemplateParameter)
	assert.Equal(t, "adminPassword", admin.Name.Data)
	assert.Equal(t, "securestring", admin.Type.Data)
	assert.True(t, admin.Secure.Data, "securestring must be flagged secure")
	assert.Equal(t, []any{"rotate-me", "rotate-me-too"}, admin.AllowedValues.Data)
	// No defaultValue declared -> null dict.
	assert.Nil(t, admin.DefaultValue.Data)
	assert.NotNil(t, admin.Metadata.Data, "metadata block should be present")

	instance := params[1].(*mqlBicepTemplateParameter)
	assert.Equal(t, "instanceCount", instance.Name.Data)
	assert.Equal(t, "int", instance.Type.Data)
	assert.False(t, instance.Secure.Data)
	assert.Equal(t, float64(3), instance.DefaultValue.Data)
	assert.Empty(t, instance.AllowedValues.Data)
}

func TestARMTemplateVariables(t *testing.T) {
	runtime := armTemplateRuntime(t)
	tmpl := armTemplate(t, runtime)

	vars, err := tmpl.variables()
	require.NoError(t, err)
	require.Len(t, vars, 1)

	v := vars[0].(*mqlBicepTemplateVariable)
	assert.Equal(t, "storageName", v.Name.Data)
	assert.Equal(t, "stcontoso001", v.Value.Data)
}

func TestARMTemplateOutputs(t *testing.T) {
	runtime := armTemplateRuntime(t)
	tmpl := armTemplate(t, runtime)

	outputs, err := tmpl.outputs()
	require.NoError(t, err)
	require.Len(t, outputs, 1)

	o := outputs[0].(*mqlBicepTemplateOutput)
	assert.Equal(t, "storageId", o.Name.Data)
	assert.Equal(t, "string", o.Type.Data)
	assert.Equal(t, "[resourceId('Microsoft.Storage/storageAccounts', variables('storageName'))]", o.Value.Data)
}

func TestARMTemplateResources(t *testing.T) {
	runtime := armTemplateRuntime(t)
	tmpl := armTemplate(t, runtime)

	resources, err := tmpl.resources()
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "Microsoft.Storage/storageAccounts", resources[0].(*mqlBicepTemplateResource).Type.Data)
}

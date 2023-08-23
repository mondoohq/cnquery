// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/terraform/connection"
	"go.mondoo.com/cnquery/types"
)

func (t *mqlTerraformState) id() (string, error) {
	return "terraform.state", nil
}

type mqlTerraformStateInternal struct {
	output plugin.TValue[*connection.Output]
}

type mqlTerraformStateModulInternal struct {
	module plugin.TValue[*connection.Module]
}

func (t *mqlTerraformState) outputs() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	state, err := conn.State()
	if err != nil {
		return nil, err
	}

	if state.Values == nil {
		return nil, nil
	}

	var list []interface{}
	for k := range state.Values.Outputs {

		output := state.Values.Outputs[k]

		r, err := CreateResource(t.MqlRuntime, "terraform.state.output", map[string]*llx.RawData{
			"identifier": llx.StringData(k),
			"sensitive":  llx.BoolData(output.Sensitive),
		})
		if err != nil {
			return nil, err
		}

		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTerraformState) rootModule() (*mqlTerraformStateModule, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	state, err := conn.State()
	if err != nil {
		return nil, err
	}

	if state.Values == nil {
		return nil, nil
	}

	r, err := newMqlModule(t.MqlRuntime, state.Values.RootModule)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t *mqlTerraformState) modules() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	providerState, err := conn.State()
	if err != nil {
		return nil, err
	}

	if providerState.Values == nil {
		return nil, nil
	}

	// resolve all tfstate modules
	moduleList := []*connection.Module{}
	moduleList = append(moduleList, providerState.Values.RootModule)
	providerState.Values.RootModule.WalkChildModules(func(m *connection.Module) {
		moduleList = append(moduleList, m)
	})

	// convert module list to mql resources
	list := []interface{}{}
	for i := range moduleList {
		r, err := newMqlModule(t.MqlRuntime, moduleList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTerraformState) resources() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	providerState, err := conn.State()
	if err != nil {
		return nil, err
	}

	if providerState.Values == nil {
		return nil, nil
	}

	// resolve all tfstate resources, to achieve this we need to walk all modules
	resourceList := []*connection.Resource{}

	resourceList = append(resourceList, providerState.Values.RootModule.Resources...)
	providerState.Values.RootModule.WalkChildModules(func(m *connection.Module) {
		resourceList = append(resourceList, m.Resources...)
	})

	// convert module list to mql resources
	list := []interface{}{}
	for i := range resourceList {
		r, err := newMqlResource(t.MqlRuntime, resourceList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

type mqlTerraformStateOutputInternal struct {
	output plugin.TValue[*connection.Output]
}

func (t *mqlTerraformStateOutput) id() (string, error) {
	id := t.Identifier
	return "terraform.state.output/identifier/" + id.Data, nil
}

func (t *mqlTerraformStateOutput) value() (interface{}, error) {
	var output *connection.Output
	if t.output.State == plugin.StateIsSet {
		output = t.output.Data
	}
	// FIXME: What happens if not set?

	var value interface{}
	if err := json.Unmarshal([]byte(output.Value), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (t mqlTerraformStateOutput) compute_type() (interface{}, error) {
	var output *connection.Output
	if t.output.State == plugin.StateIsSet {
		output = t.output.Data
	}
	// FIXME: What happens if not set?

	var typ interface{}
	if err := json.Unmarshal([]byte(output.Type), &typ); err != nil {
		return nil, err
	}
	return typ, nil
}

func (t *mqlTerraformStateModule) id() (string, error) {
	address := t.Address

	name := "terraform.module"
	if address.Data != "" {
		name += "/address/" + address.Data
	}

	return name, nil
}

type mqlTerraformStateModuleInternal struct {
	module plugin.TValue[*connection.Module]
}

func (t *mqlTerraformStateModule) resources() ([]interface{}, error) {
	var module *connection.Module
	if t.module.State == plugin.StateIsSet {
		module = t.module.Data
	}
	// FIXME: What happens if not set?

	var list []interface{}
	for i := range module.Resources {
		resource := module.Resources[i]
		r, err := newMqlResource(t.MqlRuntime, resource)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func newMqlModule(runtime *plugin.Runtime, module *connection.Module) (*mqlTerraformStateModule, error) {
	r, err := CreateResource(runtime, "terraform.state.module", map[string]*llx.RawData{
		"address": llx.StringData(module.Address),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlTerraformStateModule), nil
}

func newMqlResource(runtime *plugin.Runtime, resource *connection.Resource) (plugin.Resource, error) {
	r, err := CreateResource(runtime, "terraform.state.resource", map[string]*llx.RawData{
		"address":       llx.StringData(resource.Address),
		"name":          llx.StringData(resource.Name),
		"mode":          llx.StringData(resource.Mode),
		"type":          llx.StringData(resource.Type),
		"providerName":  llx.StringData(resource.ProviderName),
		"schemaVersion": llx.IntData(int64(resource.SchemaVersion)),
		"values":        llx.MapData(resource.AttributeValues, types.Any),
		"dependsOn":     llx.ArrayData(convert.SliceAnyToInterface[string](resource.DependsOn), types.String),
		"tainted":       llx.BoolData(resource.Tainted),
		"deposedKey":    llx.StringData(resource.DeposedKey),
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t *mqlTerraformStateModule) childModules() ([]interface{}, error) {
	var module *connection.Module
	if t.module.State == plugin.StateIsSet {
		module = t.module.Data
	}
	// FIXME: What happens if not set?

	var list []interface{}
	for i := range module.ChildModules {
		r, err := newMqlModule(t.MqlRuntime, module.ChildModules[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTerraformStateResource) id() (string, error) {
	address := t.Address

	name := "terraform.state.resource"
	if address.Data != "" {
		name += "/address/" + address.Data
	}

	return name, nil
}

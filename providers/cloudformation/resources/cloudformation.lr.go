// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by resources. DO NOT EDIT.

package resources

import (
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/types"
)

var resourceFactories map[string]plugin.ResourceFactory

func init() {
	resourceFactories = map[string]plugin.ResourceFactory {
		"cloudformation.template": {
			Init: initCloudformationTemplate,
			Create: createCloudformationTemplate,
		},
		"cloudformation.resource": {
			// to override args, implement: initCloudformationResource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createCloudformationResource,
		},
		"cloudformation.output": {
			// to override args, implement: initCloudformationOutput(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createCloudformationOutput,
		},
	}
}

// NewResource is used by the runtime of this plugin to create new resources.
// Its arguments may be provided by users. This function is generally not
// used by initializing resources from recordings or from lists.
func NewResource(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
	f, ok := resourceFactories[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in this provider")
	}

	if f.Init != nil {
		cargs, res, err := f.Init(runtime, args)
		if err != nil {
			return res, err
		}

		if res != nil {
			id := name+"\x00"+res.MqlID()
			if x, ok := runtime.Resources.Get(id); ok {
				return x, nil
			}
			runtime.Resources.Set(id, res)
			return res, nil
		}

		args = cargs
	}

	res, err := f.Create(runtime, args)
	if err != nil {
		return nil, err
	}

	id := name+"\x00"+res.MqlID()
	if x, ok := runtime.Resources.Get(id); ok {
		return x, nil
	}

	runtime.Resources.Set(id, res)
	return res, nil
}

// CreateResource is used by the runtime of this plugin to create resources.
// Its arguments must be complete and pre-processed. This method is used
// for initializing resources from recordings or from lists.
func CreateResource(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
	f, ok := resourceFactories[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in this provider")
	}

	res, err := f.Create(runtime, args)
	if err != nil {
		return nil, err
	}

	id := name+"\x00"+res.MqlID()
	if x, ok := runtime.Resources.Get(id); ok {
		return x, nil
	}

	runtime.Resources.Set(id, res)
	return res, nil
}

var getDataFields = map[string]func(r plugin.Resource) *plugin.DataRes{
	"cloudformation.template.version": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetVersion()).ToDataRes(types.String)
	},
	"cloudformation.template.transform": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetTransform()).ToDataRes(types.Array(types.String))
	},
	"cloudformation.template.description": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetDescription()).ToDataRes(types.String)
	},
	"cloudformation.template.mappings": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetMappings()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.template.globals": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetGlobals()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.template.parameters": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetParameters()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.template.metadata": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetMetadata()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.template.conditions": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetConditions()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.template.resources": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetResources()).ToDataRes(types.Array(types.Resource("cloudformation.resource")))
	},
	"cloudformation.template.outputs": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetOutputs()).ToDataRes(types.Array(types.Resource("cloudformation.output")))
	},
	"cloudformation.template.types": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationTemplate).GetTypes()).ToDataRes(types.Array(types.String))
	},
	"cloudformation.resource.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationResource).GetName()).ToDataRes(types.String)
	},
	"cloudformation.resource.type": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationResource).GetType()).ToDataRes(types.String)
	},
	"cloudformation.resource.condition": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationResource).GetCondition()).ToDataRes(types.String)
	},
	"cloudformation.resource.documentation": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationResource).GetDocumentation()).ToDataRes(types.String)
	},
	"cloudformation.resource.attributes": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationResource).GetAttributes()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.resource.properties": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationResource).GetProperties()).ToDataRes(types.Map(types.String, types.Dict))
	},
	"cloudformation.output.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationOutput).GetName()).ToDataRes(types.String)
	},
	"cloudformation.output.properties": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCloudformationOutput).GetProperties()).ToDataRes(types.Map(types.String, types.Dict))
	},
}

func GetData(resource plugin.Resource, field string, args map[string]*llx.RawData) *plugin.DataRes {
	f, ok := getDataFields[resource.MqlName()+"."+field]
	if !ok {
		return &plugin.DataRes{Error: "cannot find '" + field + "' in resource '" + resource.MqlName() + "'"}
	}

	return f(resource)
}

var setDataFields = map[string]func(r plugin.Resource, v *llx.RawData) bool {
	"cloudformation.template.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlCloudformationTemplate).__id, ok = v.Value.(string)
			return
		},
	"cloudformation.template.version": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Version, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.template.transform": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Transform, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.description": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Description, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.template.mappings": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Mappings, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.globals": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Globals, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.parameters": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Parameters, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.metadata": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Metadata, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.conditions": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Conditions, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.resources": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Resources, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.outputs": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Outputs, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.template.types": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationTemplate).Types, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.resource.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlCloudformationResource).__id, ok = v.Value.(string)
			return
		},
	"cloudformation.resource.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationResource).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.resource.type": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationResource).Type, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.resource.condition": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationResource).Condition, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.resource.documentation": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationResource).Documentation, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.resource.attributes": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationResource).Attributes, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.resource.properties": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationResource).Properties, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
	"cloudformation.output.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlCloudformationOutput).__id, ok = v.Value.(string)
			return
		},
	"cloudformation.output.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationOutput).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"cloudformation.output.properties": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCloudformationOutput).Properties, ok = plugin.RawToTValue[map[string]interface{}](v.Value, v.Error)
		return
	},
}

func SetData(resource plugin.Resource, field string, val *llx.RawData) error {
	f, ok := setDataFields[resource.MqlName() + "." + field]
	if !ok {
		return errors.New("[cloudformation] cannot set '"+field+"' in resource '"+resource.MqlName()+"', field not found")
	}

	if ok := f(resource, val); !ok {
		return errors.New("[cloudformation] cannot set '"+field+"' in resource '"+resource.MqlName()+"', type does not match")
	}
	return nil
}

func SetAllData(resource plugin.Resource, args map[string]*llx.RawData) error {
	var err error
	for k, v := range args {
		if err = SetData(resource, k, v); err != nil {
			return err
		}
	}
	return nil
}

// mqlCloudformationTemplate for the cloudformation.template resource
type mqlCloudformationTemplate struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlCloudformationTemplateInternal it will be used here
	Version plugin.TValue[string]
	Transform plugin.TValue[[]interface{}]
	Description plugin.TValue[string]
	Mappings plugin.TValue[map[string]interface{}]
	Globals plugin.TValue[map[string]interface{}]
	Parameters plugin.TValue[map[string]interface{}]
	Metadata plugin.TValue[map[string]interface{}]
	Conditions plugin.TValue[map[string]interface{}]
	Resources plugin.TValue[[]interface{}]
	Outputs plugin.TValue[[]interface{}]
	Types plugin.TValue[[]interface{}]
}

// createCloudformationTemplate creates a new instance of this resource
func createCloudformationTemplate(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlCloudformationTemplate{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("cloudformation.template", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlCloudformationTemplate) MqlName() string {
	return "cloudformation.template"
}

func (c *mqlCloudformationTemplate) MqlID() string {
	return c.__id
}

func (c *mqlCloudformationTemplate) GetVersion() *plugin.TValue[string] {
	return &c.Version
}

func (c *mqlCloudformationTemplate) GetTransform() *plugin.TValue[[]interface{}] {
	return &c.Transform
}

func (c *mqlCloudformationTemplate) GetDescription() *plugin.TValue[string] {
	return &c.Description
}

func (c *mqlCloudformationTemplate) GetMappings() *plugin.TValue[map[string]interface{}] {
	return plugin.GetOrCompute[map[string]interface{}](&c.Mappings, func() (map[string]interface{}, error) {
		return c.mappings()
	})
}

func (c *mqlCloudformationTemplate) GetGlobals() *plugin.TValue[map[string]interface{}] {
	return plugin.GetOrCompute[map[string]interface{}](&c.Globals, func() (map[string]interface{}, error) {
		return c.globals()
	})
}

func (c *mqlCloudformationTemplate) GetParameters() *plugin.TValue[map[string]interface{}] {
	return plugin.GetOrCompute[map[string]interface{}](&c.Parameters, func() (map[string]interface{}, error) {
		return c.parameters()
	})
}

func (c *mqlCloudformationTemplate) GetMetadata() *plugin.TValue[map[string]interface{}] {
	return plugin.GetOrCompute[map[string]interface{}](&c.Metadata, func() (map[string]interface{}, error) {
		return c.metadata()
	})
}

func (c *mqlCloudformationTemplate) GetConditions() *plugin.TValue[map[string]interface{}] {
	return plugin.GetOrCompute[map[string]interface{}](&c.Conditions, func() (map[string]interface{}, error) {
		return c.conditions()
	})
}

func (c *mqlCloudformationTemplate) GetResources() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Resources, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("cloudformation.template", c.__id, "resources")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.resources()
	})
}

func (c *mqlCloudformationTemplate) GetOutputs() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Outputs, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("cloudformation.template", c.__id, "outputs")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.outputs()
	})
}

func (c *mqlCloudformationTemplate) GetTypes() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Types, func() ([]interface{}, error) {
		return c.types()
	})
}

// mqlCloudformationResource for the cloudformation.resource resource
type mqlCloudformationResource struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlCloudformationResourceInternal it will be used here
	Name plugin.TValue[string]
	Type plugin.TValue[string]
	Condition plugin.TValue[string]
	Documentation plugin.TValue[string]
	Attributes plugin.TValue[map[string]interface{}]
	Properties plugin.TValue[map[string]interface{}]
}

// createCloudformationResource creates a new instance of this resource
func createCloudformationResource(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlCloudformationResource{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("cloudformation.resource", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlCloudformationResource) MqlName() string {
	return "cloudformation.resource"
}

func (c *mqlCloudformationResource) MqlID() string {
	return c.__id
}

func (c *mqlCloudformationResource) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlCloudformationResource) GetType() *plugin.TValue[string] {
	return &c.Type
}

func (c *mqlCloudformationResource) GetCondition() *plugin.TValue[string] {
	return &c.Condition
}

func (c *mqlCloudformationResource) GetDocumentation() *plugin.TValue[string] {
	return &c.Documentation
}

func (c *mqlCloudformationResource) GetAttributes() *plugin.TValue[map[string]interface{}] {
	return &c.Attributes
}

func (c *mqlCloudformationResource) GetProperties() *plugin.TValue[map[string]interface{}] {
	return &c.Properties
}

// mqlCloudformationOutput for the cloudformation.output resource
type mqlCloudformationOutput struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlCloudformationOutputInternal it will be used here
	Name plugin.TValue[string]
	Properties plugin.TValue[map[string]interface{}]
}

// createCloudformationOutput creates a new instance of this resource
func createCloudformationOutput(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlCloudformationOutput{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("cloudformation.output", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlCloudformationOutput) MqlName() string {
	return "cloudformation.output"
}

func (c *mqlCloudformationOutput) MqlID() string {
	return c.__id
}

func (c *mqlCloudformationOutput) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlCloudformationOutput) GetProperties() *plugin.TValue[map[string]interface{}] {
	return &c.Properties
}

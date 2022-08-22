package core

import (
	"encoding/json"
	"errors"

	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/tfstate"
	"go.mondoo.io/mondoo/resources"
)

func tfstateProvider(t providers.Transport) (*tfstate.Provider, error) {
	gt, ok := t.(*tfstate.Provider)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this transport")
	}
	return gt, nil
}

func (t *mqlTfstate) id() (string, error) {
	return "tfstate", nil
}

func (t *mqlTfstate) init(args *resources.Args) (*resources.Args, Tfstate, error) {
	tfstateProvider, err := tfstateProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	state, err := tfstateProvider.State()
	if err != nil {
		return nil, nil, err
	}

	(*args)["formatVersion"] = state.FormatVersion
	(*args)["terraformVersion"] = state.TerraformVersion

	return args, nil, nil
}

func (t *mqlTfstate) GetOutputs() ([]interface{}, error) {
	provider, err := tfstateProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	state, err := provider.State()
	if err != nil {
		return nil, err
	}

	if state.Values == nil {
		return nil, nil
	}

	var list []interface{}
	for k := range state.Values.Outputs {

		output := state.Values.Outputs[k]

		r, err := t.MotorRuntime.CreateResource("tfstate.output",
			"identifier", k,
			"sensitive", output.Sensitive,
		)
		if err != nil {
			return nil, err
		}
		// store output in cache
		r.MqlResource().Cache.Store("_output", &resources.CacheEntry{Data: output})

		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTfstate) GetRootModule() (interface{}, error) {
	provider, err := tfstateProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	state, err := provider.State()
	if err != nil {
		return nil, err
	}

	if state.Values == nil {
		return nil, nil
	}

	r, err := newMqlModule(t.MotorRuntime, state.Values.RootModule)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t *mqlTfstate) GetModules() (interface{}, error) {
	provider, err := tfstateProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	state, err := provider.State()
	if err != nil {
		return nil, err
	}

	if state.Values == nil {
		return nil, nil
	}

	// resolve all tfstate modules
	moduleList := []*tfstate.Module{}
	moduleList = append(moduleList, state.Values.RootModule)
	state.Values.RootModule.WalkChildModules(func(m *tfstate.Module) {
		moduleList = append(moduleList, m)
	})

	// convert module list to mql resources
	list := []interface{}{}
	for i := range moduleList {
		r, err := newMqlModule(t.MotorRuntime, moduleList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTfstateOutput) id() (string, error) {
	id, err := t.Identifier()
	if err != nil {
		return "", err
	}
	return "tfstateoutput/identifier/" + id, nil
}

func (t *mqlTfstateOutput) init(args *resources.Args) (*resources.Args, TfstateOutput, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	// check if identifier is there
	nameRaw := (*args)["identifier"]
	if nameRaw != nil {
		name := nameRaw.(string)
		obj, err := t.MotorRuntime.CreateResource("tfstate")
		if err != nil {
			return nil, nil, err
		}
		tfstate := obj.(Tfstate)

		outputs, err := tfstate.Outputs()
		if err != nil {
			return nil, nil, err
		}

		for i := range outputs {
			o := outputs[i].(TfstateOutput)
			id, _ := o.Identifier()
			if id == name {
				return nil, o, nil
			}
		}
	}

	return args, nil, nil
}

func (t *mqlTfstateOutput) GetValue() (interface{}, error) {
	c, ok := t.MqlResource().Cache.Load("_output")
	if !ok {
		return nil, errors.New("cannot get output cache")
	}
	output := c.Data.(*tfstate.Output)

	var value interface{}
	if err := json.Unmarshal([]byte(output.Value), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (t *mqlTfstateOutput) GetType() (interface{}, error) {
	c, ok := t.MqlResource().Cache.Load("_output")
	if !ok {
		return nil, errors.New("cannot get output cache")
	}
	output := c.Data.(*tfstate.Output)

	var typ interface{}
	if err := json.Unmarshal([]byte(output.Type), &typ); err != nil {
		return nil, err
	}
	return typ, nil
}

func (t *mqlTfstateModule) id() (string, error) {
	address, err := t.Address()
	if err != nil {
		return "", err
	}

	name := "tfmodule"
	if address != "" {
		name += "/address/" + address
	}

	return name, nil
}

func (t *mqlTfstateModule) init(args *resources.Args) (*resources.Args, TfstateModule, error) {
	// check if identifier is there
	nameRaw := (*args)["address"]
	if nameRaw != nil {
		return args, nil, nil
	}

	idRaw := (*args)["identifier"]
	if idRaw != nil {
		identifier := idRaw.(string)
		obj, err := t.MotorRuntime.CreateResource("tfstate")
		if err != nil {
			return nil, nil, err
		}
		tfstate := obj.(Tfstate)

		modules, err := tfstate.Modules()
		if err != nil {
			return nil, nil, err
		}

		for i := range modules {
			o := modules[i].(TfstateModule)
			id, _ := o.Address()
			if id == identifier {
				return nil, o, nil
			}
		}
		delete(*args, "identifier")
	}

	return args, nil, nil
}

func (t *mqlTfstateModule) GetResources() ([]interface{}, error) {
	c, ok := t.MqlResource().Cache.Load("_module")
	if !ok {
		return nil, errors.New("cannot get module cache")
	}
	module := c.Data.(*tfstate.Module)

	var list []interface{}
	for i := range module.Resources {

		resource := module.Resources[i]

		r, err := t.MotorRuntime.CreateResource("tfstate.resource",
			"address", resource.Address,
			"name", resource.Name,
			"mode", resource.Mode,
			"type", resource.Type,
			"providerName", resource.ProviderName,
			"schemaVersion", int64(resource.SchemaVersion),
			"values", resource.AttributeValues,
			"dependsOn", StrSliceToInterface(resource.DependsOn),
			"tainted", resource.Tainted,
			"deposedKey", resource.DeposedKey,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func newMqlModule(runtime *resources.Runtime, module *tfstate.Module) (resources.ResourceType, error) {
	r, err := runtime.CreateResource("tfstate.module",
		"address", module.Address,
	)
	if err != nil {
		return nil, err
	}
	// store module in cache
	r.MqlResource().Cache.Store("_module", &resources.CacheEntry{Data: module})
	return r, nil
}

func (t *mqlTfstateModule) GetChildModules() ([]interface{}, error) {
	c, ok := t.MqlResource().Cache.Load("_module")
	if !ok {
		return nil, errors.New("cannot get module cache")
	}
	module := c.Data.(*tfstate.Module)

	var list []interface{}
	for i := range module.ChildModules {
		r, err := newMqlModule(t.MotorRuntime, module.ChildModules[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTfstateResource) id() (string, error) {
	address, err := t.Address()
	if err != nil {
		return "", err
	}

	name := "tfstateresource"
	if address != "" {
		name += "/address/" + address
	}

	return address, nil
}

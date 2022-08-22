package terraform

import (
	"encoding/json"
	"errors"

	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/tfstate"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func tfstateProvider(t providers.Transport) (*tfstate.Provider, error) {
	gt, ok := t.(*tfstate.Provider)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this transport")
	}
	return gt, nil
}

func (t *mqlTerraformState) id() (string, error) {
	return "terraform.state", nil
}

func (t *mqlTerraformState) init(args *resources.Args) (*resources.Args, TerraformState, error) {
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

func (t *mqlTerraformState) GetOutputs() ([]interface{}, error) {
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

		r, err := t.MotorRuntime.CreateResource("terraform.state.output",
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

func (t *mqlTerraformState) GetRootModule() (interface{}, error) {
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

func (t *mqlTerraformState) GetModules() (interface{}, error) {
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

func (t *mqlTerraformStateOutput) id() (string, error) {
	id, err := t.Identifier()
	if err != nil {
		return "", err
	}
	return "terraform.state.output/identifier/" + id, nil
}

func (t *mqlTerraformStateOutput) init(args *resources.Args) (*resources.Args, TerraformStateOutput, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	// check if identifier is there
	nameRaw := (*args)["identifier"]
	if nameRaw != nil {
		name := nameRaw.(string)
		obj, err := t.MotorRuntime.CreateResource("terraform.state")
		if err != nil {
			return nil, nil, err
		}
		tfstate := obj.(TerraformState)

		outputs, err := tfstate.Outputs()
		if err != nil {
			return nil, nil, err
		}

		for i := range outputs {
			o := outputs[i].(TerraformStateOutput)
			id, _ := o.Identifier()
			if id == name {
				return nil, o, nil
			}
		}
	}

	return args, nil, nil
}

func (t *mqlTerraformStateOutput) GetValue() (interface{}, error) {
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

func (t mqlTerraformStateOutput) GetType() (interface{}, error) {
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

func (t *mqlTerraformStateModule) id() (string, error) {
	address, err := t.Address()
	if err != nil {
		return "", err
	}

	name := "terraform.module"
	if address != "" {
		name += "/address/" + address
	}

	return name, nil
}

func (t *mqlTerraformStateModule) init(args *resources.Args) (*resources.Args, TerraformStateModule, error) {
	// check if identifier is there
	nameRaw := (*args)["address"]
	if nameRaw != nil {
		return args, nil, nil
	}

	idRaw := (*args)["identifier"]
	if idRaw != nil {
		identifier := idRaw.(string)
		obj, err := t.MotorRuntime.CreateResource("terraform.state")
		if err != nil {
			return nil, nil, err
		}
		tfstate := obj.(TerraformState)

		modules, err := tfstate.Modules()
		if err != nil {
			return nil, nil, err
		}

		for i := range modules {
			o := modules[i].(TerraformStateModule)
			id, _ := o.Address()
			if id == identifier {
				return nil, o, nil
			}
		}
		delete(*args, "identifier")
	}

	return args, nil, nil
}

func (t *mqlTerraformStateModule) GetResources() ([]interface{}, error) {
	c, ok := t.MqlResource().Cache.Load("_module")
	if !ok {
		return nil, errors.New("cannot get module cache")
	}
	module := c.Data.(*tfstate.Module)

	var list []interface{}
	for i := range module.Resources {

		resource := module.Resources[i]

		r, err := t.MotorRuntime.CreateResource("terraform.state.resource",
			"address", resource.Address,
			"name", resource.Name,
			"mode", resource.Mode,
			"type", resource.Type,
			"providerName", resource.ProviderName,
			"schemaVersion", int64(resource.SchemaVersion),
			"values", resource.AttributeValues,
			"dependsOn", core.StrSliceToInterface(resource.DependsOn),
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
	r, err := runtime.CreateResource("terraform.state.module",
		"address", module.Address,
	)
	if err != nil {
		return nil, err
	}
	// store module in cache
	r.MqlResource().Cache.Store("_module", &resources.CacheEntry{Data: module})
	return r, nil
}

func (t *mqlTerraformStateModule) GetChildModules() ([]interface{}, error) {
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

func (t *mqlTerraformStateResource) id() (string, error) {
	address, err := t.Address()
	if err != nil {
		return "", err
	}

	name := "terraform.state.resource"
	if address != "" {
		name += "/address/" + address
	}

	return address, nil
}

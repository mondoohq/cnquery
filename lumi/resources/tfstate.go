package resources

import (
	"encoding/json"
	"errors"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/tfstate"
)

func tfstateProvider(t providers.Transport) (*tfstate.Provider, error) {
	gt, ok := t.(*tfstate.Provider)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this transport")
	}
	return gt, nil
}

func (t *lumiTfstate) id() (string, error) {
	return "tfstate", nil
}

func (t *lumiTfstate) init(args *lumi.Args) (*lumi.Args, Tfstate, error) {
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

func (t *lumiTfstate) GetOutputs() ([]interface{}, error) {
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
		r.LumiResource().Cache.Store("_output", &lumi.CacheEntry{Data: output})

		list = append(list, r)
	}

	return list, nil
}

func (t *lumiTfstate) GetRootModule() (interface{}, error) {
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

	r, err := newLumiModule(t.MotorRuntime, state.Values.RootModule)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t *lumiTfstate) GetModules() (interface{}, error) {
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
		r, err := newLumiModule(t.MotorRuntime, moduleList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *lumiTfstateOutput) id() (string, error) {
	id, err := t.Identifier()
	if err != nil {
		return "", err
	}
	return "tfstateoutput/identifier/" + id, nil
}

func (t *lumiTfstateOutput) init(args *lumi.Args) (*lumi.Args, TfstateOutput, error) {
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

func (t *lumiTfstateOutput) GetValue() (interface{}, error) {
	c, ok := t.LumiResource().Cache.Load("_output")
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

func (t *lumiTfstateOutput) GetType() (interface{}, error) {
	c, ok := t.LumiResource().Cache.Load("_output")
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

func (t *lumiTfstateModule) id() (string, error) {
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

func (t *lumiTfstateModule) init(args *lumi.Args) (*lumi.Args, TfstateModule, error) {
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

func (t *lumiTfstateModule) GetResources() ([]interface{}, error) {
	c, ok := t.LumiResource().Cache.Load("_module")
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
			"dependsOn", sliceInterface(resource.DependsOn),
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

func newLumiModule(runtime *lumi.Runtime, module *tfstate.Module) (lumi.ResourceType, error) {
	r, err := runtime.CreateResource("tfstate.module",
		"address", module.Address,
	)
	if err != nil {
		return nil, err
	}
	// store module in cache
	r.LumiResource().Cache.Store("_module", &lumi.CacheEntry{Data: module})
	return r, nil
}

func (t *lumiTfstateModule) GetChildModules() ([]interface{}, error) {
	c, ok := t.LumiResource().Cache.Load("_module")
	if !ok {
		return nil, errors.New("cannot get module cache")
	}
	module := c.Data.(*tfstate.Module)

	var list []interface{}
	for i := range module.ChildModules {
		r, err := newLumiModule(t.MotorRuntime, module.ChildModules[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *lumiTfstateResource) id() (string, error) {
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

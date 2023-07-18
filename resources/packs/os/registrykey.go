package os

import (
	"errors"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/os/windows"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

func (k *mqlRegistrykey) id() (string, error) {
	return k.Path()
}

func (k *mqlRegistrykey) GetExists() (bool, error) {
	path, err := k.Path()
	if err != nil {
		return false, err
	}

	// if we are running locally on windows, we can use native api
	_, ok := k.MotorRuntime.Motor.Provider.(*local.Provider)
	if ok && runtime.GOOS == "windows" {
		items, err := windows.GetNativeRegistryKeyItems(path)
		if err == nil && len(items) > 0 {
			return true, nil
		}
		std, ok := status.FromError(err)
		if ok && std.Code() == codes.NotFound {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}

	script := powershell.Encode(windows.GetRegistryKeyItemScript(path))
	mqlCmd, err := k.MotorRuntime.CreateResource("command", "command", script)
	if err != nil {
		log.Error().Err(err).Msg("could not create resource")
		return false, err
	}
	cmd := mqlCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if err != nil {
		return false, err
	}
	if exitcode != 0 {
		stderr, err := cmd.Stderr()
		// this would be an expected error and would ensure that we do not throw an error on windows systems
		// TODO: revisit how this is handled for non-english systems
		if err == nil && (strings.Contains(stderr, "not exist") || strings.Contains(stderr, "ObjectNotFound")) {
			return false, nil
		}

		return false, errors.New("could not retrieve registry key")
	}
	return true, nil
}

// GetEntries returns a list of registry key property resources
func (k *mqlRegistrykey) getEntries() ([]windows.RegistryKeyItem, error) {
	path, err := k.Path()
	if err != nil {
		return nil, err
	}

	// if we are running locally on windows, we can use native api
	_, ok := k.MotorRuntime.Motor.Provider.(*local.Provider)
	if ok && runtime.GOOS == "windows" {
		return windows.GetNativeRegistryKeyItems(path)
	}

	// parse the output of the powershell script
	script := powershell.Encode(windows.GetRegistryKeyItemScript(path))
	mqlCmd, err := k.MotorRuntime.CreateResource("command", "command", script)
	if err != nil {
		return nil, err
	}
	cmd := mqlCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if err != nil {
		return nil, err
	}
	if exitcode != 0 {
		return nil, errors.New("could not retrieve registry key")
	}

	stdout, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	return windows.ParsePowershellRegistryKeyItems(strings.NewReader(stdout))
}

// Deprecated: GetProperties returns the properties of a registry key
// This function is deprecated and will be removed in a future release
func (k *mqlRegistrykey) GetProperties() (map[string]interface{}, error) {
	res := map[string]interface{}{}

	entries, err := k.getEntries()
	if err != nil {
		return nil, err
	}

	for i := range entries {
		rkey := entries[i]
		res[rkey.Key] = rkey.String()
	}

	return res, nil
}

// GetItems returns a list of registry key property resources
func (k *mqlRegistrykey) GetItems() ([]interface{}, error) {
	entries, err := k.getEntries()
	if err != nil {
		return nil, err
	}

	path, err := k.Path()
	if err != nil {
		return nil, err
	}

	// create MQL mount entry resources for each mount
	items := make([]interface{}, len(entries))
	for i, entry := range entries {
		mqlRegistryPropertyEntry, err := k.MotorRuntime.CreateResource("registrykey.property",
			"path", path,
			"name", entry.Key,
			"value", entry.String(),
			"type", entry.Kind(),
			"data", entry.GetRawValue(),
			"exists", true,
		)
		if err != nil {
			return nil, err
		}

		items[i] = mqlRegistryPropertyEntry.(RegistrykeyProperty)
	}

	return items, nil
}

func (k *mqlRegistrykey) GetChildren() ([]interface{}, error) {
	res := []interface{}{}

	path, err := k.Path()
	if err != nil {
		return nil, err
	}

	var children []windows.RegistryKeyChild
	// if we are running locally on windows, we can use native api
	_, ok := k.MotorRuntime.Motor.Provider.(*local.Provider)
	if ok && runtime.GOOS == "windows" {
		children, err = windows.GetNativeRegistryKeyChildren(path)
		if err != nil {
			return nil, err
		}
	} else {
		// parse powershell script
		script := powershell.Encode(windows.GetRegistryKeyChildItemsScript(path))
		mqlCmd, err := k.MotorRuntime.CreateResource("command", "command", script)
		if err != nil {
			return res, err
		}
		cmd := mqlCmd.(Command)
		exitcode, err := cmd.Exitcode()
		if err != nil {
			return nil, err
		}
		if exitcode != 0 {
			return nil, errors.New("could not retrieve registry key")
		}

		stdout, err := cmd.Stdout()
		if err != nil {
			return res, err
		}
		children, err = windows.ParsePowershellRegistryKeyChildren(strings.NewReader(stdout))
		if err != nil {
			return nil, err
		}
	}

	for i := range children {
		child := children[i]
		res = append(res, child.Path)
	}

	return res, nil
}

func (p *mqlRegistrykeyProperty) id() (string, error) {
	path, err := p.Path()
	if err != nil {
		return "", err
	}

	name, err := p.Name()
	if err != nil {
		return "", err
	}

	return path + " - " + name, nil
}

func (p *mqlRegistrykeyProperty) init(args *resources.Args) (*resources.Args, RegistrykeyProperty, error) {
	pathRaw := (*args)["path"]
	if pathRaw == nil {
		return args, nil, nil
	}

	path, ok := pathRaw.(string)
	if !ok {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.(string)
	if !ok {
		return args, nil, nil
	}

	// if the data is set, we do not need to fetch the data first
	dataRaw := (*args)["data"]
	if dataRaw != nil {
		return args, nil, nil
	}

	// create resource here, but do not use it yet
	obj, err := p.MotorRuntime.CreateResource("registrykey", "path", path)
	if err != nil {
		return nil, nil, err
	}
	registryKey := obj.(Registrykey)

	exists, err := registryKey.Exists()
	if err != nil {
		return nil, nil, err
	}

	// set default values
	(*args)["exists"] = false
	// NOTE: we do not set a value here so that MQL throws an error when a user try to gather the data for a
	// non-existing key

	// path exists
	if exists {
		items, err := registryKey.Items()
		if err != nil {
			return nil, nil, err
		}

		for i := range items {
			property := items[i].(RegistrykeyProperty)
			itemName, err := property.Name()
			if err != nil {
				return nil, nil, err
			}

			// property exists, return it
			if strings.EqualFold(itemName, name) {
				return nil, property, nil
			}
		}
	}
	return args, nil, nil
}

func (p *mqlRegistrykeyProperty) GetExists() (bool, error) {
	// NOTE: will not be called since it will always be set in init
	return false, errors.New("could not determine if the property exists")
}

func (p *mqlRegistrykeyProperty) GetType() (string, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

func (p *mqlRegistrykeyProperty) GetData() (interface{}, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

func (p *mqlRegistrykeyProperty) GetValue() (interface{}, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

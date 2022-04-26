package resources

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (k *lumiRegistrykey) id() (string, error) {
	return k.Path()
}

func (k *lumiRegistrykey) GetExists() (bool, error) {
	path, err := k.Path()
	if err != nil {
		return false, err
	}

	script := powershell.Encode(windows.GetRegistryKeyItemScript(path))
	lumiCmd, err := k.Runtime.CreateResource("command", "command", script)
	if err != nil {
		log.Error().Err(err).Msg("could not create resource")
		return false, err
	}
	cmd := lumiCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if err != nil {
		return false, err
	}
	if exitcode != 0 {
		stderr, err := cmd.Stderr()
		// this would be an expected error and would ensure that we do not throw an error on windows systems
		// TODO: revist how this is handled for non-english systems
		if err == nil && strings.Contains(stderr, "does not exist") {
			return false, nil
		}

		return false, errors.New("could to retrieve registry key")
	}
	return true, nil
}

func (k *lumiRegistrykey) GetProperties() (map[string]interface{}, error) {
	path, err := k.Path()
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	script := powershell.Encode(windows.GetRegistryKeyItemScript(path))
	lumiCmd, err := k.Runtime.CreateResource("command", "command", script)
	if err != nil {
		return nil, err
	}
	cmd := lumiCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if err != nil {
		return nil, err
	}
	if exitcode != 0 {
		return nil, errors.New("could to retrieve registry key")
	}

	stdout, err := cmd.Stdout()
	if err != nil {
		return res, err
	}
	entries, err := windows.ParseRegistryKeyItems(strings.NewReader(stdout))
	if err != nil {
		return nil, err
	}

	for i := range entries {
		rkey := entries[i]
		res[rkey.Key] = rkey.GetValue()
	}

	return res, nil
}

func (k *lumiRegistrykey) GetChildren() ([]interface{}, error) {
	res := []interface{}{}

	path, err := k.Path()
	if err != nil {
		return nil, err
	}

	script := powershell.Encode(windows.GetRegistryKeyChildItemsScript(path))
	lumiCmd, err := k.Runtime.CreateResource("command", "command", script)
	if err != nil {
		return res, err
	}
	cmd := lumiCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if err != nil {
		return nil, err
	}
	if exitcode != 0 {
		return nil, errors.New("could to retrieve registry key")
	}

	stdout, err := cmd.Stdout()
	if err != nil {
		return res, err
	}
	children, err := windows.ParseRegistryKeyChildren(strings.NewReader(stdout))
	if err != nil {
		return nil, err
	}

	for i := range children {
		child := children[i]
		res = append(res, child.Path)
	}

	return res, nil
}

func (p *lumiRegistrykeyProperty) id() (string, error) {
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

func (p *lumiRegistrykeyProperty) init(args *lumi.Args) (*lumi.Args, RegistrykeyProperty, error) {
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

	// create resource here, but do not use it yet
	obj, err := p.Runtime.CreateResource("registrykey", "path", path)
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
	// NOTE: we do not set a value here so that lumi throws an error when a user try to gather the data for a
	// non-existing key

	// path exists
	if exists {
		properties, err := registryKey.Properties()
		if err != nil {
			return nil, nil, err
		}

		// search for property
		for k := range properties {
			if strings.EqualFold(k, name) {
				(*args)["exists"] = true
				(*args)["value"] = properties[k].(string)
				break
			}
		}
	}
	return args, nil, nil
}

func (p *lumiRegistrykeyProperty) GetExists() (bool, error) {
	// NOTE: will not be called since it will always be set in init
	return false, errors.New("could not determine if the property exists")
}

func (p *lumiRegistrykeyProperty) GetValue() (string, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

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
		return false, err
	}
	cmd := lumiCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if exitcode == 0 && err == nil {
		return true, nil
	}

	return false, nil
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
		return res, err
	}
	cmd := lumiCmd.(Command)
	exitcode, err := cmd.Exitcode()
	if exitcode == 0 && err == nil {
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
	if exitcode == 0 && err == nil {
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

	log.Debug().Str("path", path).Msg("registrykey.property> parent exists")
	exists, err := registryKey.Exists()
	if err != nil {
		return nil, nil, err
	}

	// set default values
	(*args)["value"] = ""
	(*args)["exists"] = false

	// path exists
	if exists {
		properties, err := registryKey.Properties()
		if err != nil {
			return nil, nil, err
		}

		// search for property
		for k := range properties {
			if strings.ToLower(k) == strings.ToLower(name) {
				(*args)["exists"] = true
				(*args)["value"] = properties[k].(string)
				break
			}
		}
	}
	return args, nil, nil
}

func (p *lumiRegistrykeyProperty) GetExists() (bool, error) {
	// NOTE: will not be called since its set in the constructor
	return false, errors.New("not implemented")
}

func (p *lumiRegistrykeyProperty) GetValue() (string, error) {
	// NOTE: will not be called since its set in the constructor
	return "", errors.New("not implemented")
}

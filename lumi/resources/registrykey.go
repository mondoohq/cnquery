package resources

import (
	"strings"

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

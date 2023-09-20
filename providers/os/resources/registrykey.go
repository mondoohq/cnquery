// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"runtime"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/providers/os/resources/windows"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

func (k *mqlRegistrykey) id() (string, error) {
	return k.Path.Data, nil
}

func (k *mqlRegistrykey) exists() (bool, error) {
	// if we are running locally on windows, we can use native api
	if runtime.GOOS == "windows" {
		items, err := windows.GetNativeRegistryKeyItems(k.Path.Data)
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

	script := powershell.Encode(windows.GetRegistryKeyItemScript(k.Path.Data))
	o, err := CreateResource(k.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(script),
	})
	if err != nil {
		return false, err
	}
	cmd := o.(*mqlCommand)

	exit := cmd.GetExitcode()
	if exit.Error != nil {
		return false, exit.Error
	}
	if exit.Data != 0 {
		stderr := cmd.GetStderr()
		// this would be an expected error and would ensure that we do not throw an error on windows systems
		// TODO: revisit how this is handled for non-english systems
		if err == nil && (strings.Contains(stderr.Data, "not exist") || strings.Contains(stderr.Data, "ObjectNotFound")) {
			return false, nil
		}

		return false, errors.New("could not retrieve registry key")
	}
	return true, nil
}

// GetEntries returns a list of registry key property resources
func (k *mqlRegistrykey) getEntries() ([]windows.RegistryKeyItem, error) {
	// if we are running locally on windows, we can use native api
	if runtime.GOOS == "windows" {
		return windows.GetNativeRegistryKeyItems(k.Path.Data)
	}

	// parse the output of the powershell script
	script := powershell.Encode(windows.GetRegistryKeyItemScript(k.Path.Data))
	o, err := CreateResource(k.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(script),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	exit := cmd.GetExitcode()
	if exit.Error != nil {
		return nil, exit.Error
	}
	if exit.Data != 0 {
		return nil, errors.New("could not retrieve registry key")
	}

	stdout := cmd.GetStdout()
	if stdout.Error != nil {
		return nil, stdout.Error
	}

	return windows.ParsePowershellRegistryKeyItems(strings.NewReader(stdout.Data))
}

// Deprecated: properties returns the properties of a registry key
// This function is deprecated and will be removed in a future release
func (k *mqlRegistrykey) properties() (map[string]interface{}, error) {
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

// items returns a list of registry key property resources
func (k *mqlRegistrykey) items() ([]interface{}, error) {
	entries, err := k.getEntries()
	if err != nil {
		return nil, err
	}

	// create MQL mount entry resources for each mount
	items := make([]interface{}, len(entries))
	for i, entry := range entries {
		o, err := CreateResource(k.MqlRuntime, "registrykey.property", map[string]*llx.RawData{
			"path":   llx.StringData(k.Path.Data),
			"name":   llx.StringData(entry.Key),
			"value":  llx.StringData(entry.String()),
			"type":   llx.StringData(entry.Kind()),
			"data":   llx.DictData(entry.GetRawValue()),
			"exists": llx.BoolData(true),
		})
		if err != nil {
			return nil, err
		}

		items[i] = o.(*mqlRegistrykeyProperty)
	}

	return items, nil
}

func (k *mqlRegistrykey) children() ([]interface{}, error) {
	res := []interface{}{}

	var children []windows.RegistryKeyChild
	if runtime.GOOS == "windows" {
		var err error
		children, err = windows.GetNativeRegistryKeyChildren(k.Path.Data)
		if err != nil {
			return nil, err
		}
	} else {
		// parse powershell script
		script := powershell.Encode(windows.GetRegistryKeyChildItemsScript(k.Path.Data))
		o, err := CreateResource(k.MqlRuntime, "command", map[string]*llx.RawData{
			"command": llx.StringData(script),
		})
		if err != nil {
			return res, err
		}
		cmd := o.(*mqlCommand)
		exitcode := cmd.GetExitcode()
		if exitcode.Error != nil {
			return nil, exitcode.Error
		}
		if exitcode.Data != 0 {
			return nil, errors.New("could not retrieve registry key")
		}

		stdout := cmd.GetStdout()
		if stdout.Error != nil {
			return res, stdout.Error
		}
		children, err = windows.ParsePowershellRegistryKeyChildren(strings.NewReader(stdout.Data))
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
	return p.Path.Data + " - " + p.Name.Data, nil
}

func initRegistrykeyProperty(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	path := args["path"]
	if path == nil {
		return args, nil, nil
	}

	name := args["name"]
	if name == nil {
		return args, nil, nil
	}

	// if the data is set, we do not need to fetch the data first
	dataRaw := args["data"]
	if dataRaw != nil {
		return args, nil, nil
	}

	// create resource here, but do not use it yet
	obj, err := CreateResource(runtime, "registrykey", map[string]*llx.RawData{
		"path": path,
	})
	if err != nil {
		return nil, nil, err
	}
	key := obj.(*mqlRegistrykey)

	exists := key.GetExists()
	if err != nil {
		return nil, nil, err
	}

	// set default values
	args["exists"] = llx.BoolFalse
	// NOTE: we do not set a value here so that MQL throws an error when a user try to gather the data for a
	// non-existing key

	// path exists
	if exists.Data {
		items := key.GetItems()
		if items.Error != nil {
			return nil, nil, items.Error
		}

		for i := range items.Data {
			property := items.Data[i].(*mqlRegistrykeyProperty)
			iname := property.GetName()
			if iname.Error != nil {
				return nil, nil, iname.Error
			}

			// property exists, return it
			if strings.EqualFold(iname.Data, name.Value.(string)) {
				return nil, property, nil
			}
		}
	}
	return args, nil, nil
}

func (p *mqlRegistrykeyProperty) exists() (bool, error) {
	// NOTE: will not be called since it will always be set in init
	return false, errors.New("could not determine if the property exists")
}

func (p *mqlRegistrykeyProperty) compute_type() (string, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

func (p *mqlRegistrykeyProperty) data() (interface{}, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

func (p *mqlRegistrykeyProperty) value() (string, error) {
	// NOTE: if we reach here the value has not been set in init, therefore we return an error
	return "", errors.New("requested property does not exist")
}

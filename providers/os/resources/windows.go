// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/packages"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/v11/providers/os/resources/windows"
)

func (s *mqlWindows) computerInfo() (map[string]interface{}, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	cmd := windows.PSGetComputerInfo

	// encode the powershell command
	encodedCmd := powershell.Encode(cmd)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	// If the exit code is not 0, then we got an error and we should read stderr for details
	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve computer info: " + string(stderr))
	}

	parsedInfo, err := windows.ParseComputerInfo(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// If we have no error but OsProductType is nil, we need to run a custom command to get the info
	// For reference, see https://github.com/mondoohq/cnquery/pull/4520
	if parsedInfo["OsProductType"] == nil {
		executedCmd, err := conn.RunCommand(powershell.Encode(windows.PSGetComputerInfoCustom))
		if err != nil {
			return nil, err
		}
		if executedCmd.ExitStatus != 0 {
			stderr, err := io.ReadAll(executedCmd.Stderr)
			if err != nil {
				return nil, err
			}
			return nil, errors.New("failed to retrieve computer info: " + string(stderr))
		}
		parsedInfo, err = windows.ParseCustomComputerInfo(executedCmd.Stdout)
		if err != nil {
			return nil, err
		}
	}

	return parsedInfo, nil
}

func (wh *mqlWindowsHotfix) id() (string, error) {
	return wh.HotfixId.Data, nil
}

func initWindowsHotfix(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	nameRaw := args["hotfixId"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := NewResource(runtime, "windows", nil)
	if err != nil {
		return nil, nil, err
	}
	winResource := obj.(*mqlWindows)

	hotfixes := winResource.GetHotfixes()
	if hotfixes.Error != nil {
		return nil, nil, hotfixes.Error
	}

	for i := range hotfixes.Data {
		hf := hotfixes.Data[i].(*mqlWindowsHotfix)
		if hf.HotfixId.Data == name {
			return nil, hf, nil
		}
	}

	// if the hotfix cannot be found we return an error
	return nil, nil, errors.New("could not find hotfix " + name)
}

func (w *mqlWindows) hotfixes() ([]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query hotfixes
	encodedCmd := powershell.Encode(packages.WINDOWS_QUERY_HOTFIXES)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve hotfixes: " + string(stderr))
	}

	hotfixes, err := packages.ParseWindowsHotfixes(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert hotfixes to MQL resource
	mqlHotFixes := make([]interface{}, len(hotfixes))
	for i, hf := range hotfixes {
		mqlHotfix, err := CreateResource(w.MqlRuntime, "windows.hotfix", map[string]*llx.RawData{
			"hotfixId":    llx.StringData(hf.HotFixId),
			"caption":     llx.StringData(hf.Caption),
			"description": llx.StringData(hf.Description),
			"installedOn": llx.TimeDataPtr(hf.InstalledOnTime()),
			"installedBy": llx.StringData(hf.InstalledBy),
		})
		if err != nil {
			return nil, err
		}

		mqlHotFixes[i] = mqlHotfix
	}

	return mqlHotFixes, nil
}

func (wh *mqlWindowsFeature) id() (string, error) {
	return wh.Path.Data, nil
}

func initWindowsFeature(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := NewResource(runtime, "windows", nil)
	if err != nil {
		return nil, nil, err
	}
	winResource := obj.(*mqlWindows)

	features := winResource.GetFeatures()
	if features.Error != nil {
		return nil, nil, features.Error
	}

	for i := range features.Data {
		hf := features.Data[i].(*mqlWindowsFeature)
		if hf.Name.Data == name {
			return nil, hf, nil
		}
	}

	// if the feature cannot be found we return an error
	return nil, nil, errors.New("could not find feature " + name)
}

func (w *mqlWindows) features() ([]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query features
	encodedCmd := powershell.Encode(windows.QUERY_FEATURES)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve features: " + string(stderr))
	}

	features, err := windows.ParseWindowsFeatures(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert features to MQL resource
	mqlFeatures := make([]interface{}, len(features))
	for i, feature := range features {

		mqlFeature, err := CreateResource(w.MqlRuntime, "windows.feature", map[string]*llx.RawData{
			"path":         llx.StringData(feature.Path),
			"name":         llx.StringData(feature.Name),
			"displayName":  llx.StringData(feature.DisplayName),
			"description":  llx.StringData(feature.Description),
			"installed":    llx.BoolData(feature.Installed),
			"installState": llx.IntData(feature.InstallState),
		})
		if err != nil {
			return nil, err
		}

		mqlFeatures[i] = mqlFeature
	}

	return mqlFeatures, nil
}

func (wh *mqlWindowsServerFeature) id() (string, error) {
	return wh.Path.Data, nil
}

func initWindowsServerFeature(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := NewResource(runtime, "windows", nil)
	if err != nil {
		return nil, nil, err
	}
	winResource := obj.(*mqlWindows)

	features := winResource.GetFeatures()
	if features.Error != nil {
		return nil, nil, features.Error
	}

	for i := range features.Data {
		hf := features.Data[i].(*mqlWindowsServerFeature)
		if hf.Name.Data == name {
			return nil, hf, nil
		}
	}

	// if the feature cannot be found we return an error
	return nil, nil, errors.New("could not find feature " + name)
}

func (w *mqlWindows) serverFeatures() ([]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query features
	encodedCmd := powershell.Encode(windows.QUERY_FEATURES)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve features: " + string(stderr))
	}

	features, err := windows.ParseWindowsFeatures(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert features to MQL resource
	mqlFeatures := make([]interface{}, len(features))
	for i, feature := range features {

		mqlFeature, err := CreateResource(w.MqlRuntime, "windows.serverFeature", map[string]*llx.RawData{
			"path":         llx.StringData(feature.Path),
			"name":         llx.StringData(feature.Name),
			"displayName":  llx.StringData(feature.DisplayName),
			"description":  llx.StringData(feature.Description),
			"installed":    llx.BoolData(feature.Installed),
			"installState": llx.IntData(feature.InstallState),
		})
		if err != nil {
			return nil, err
		}

		mqlFeatures[i] = mqlFeature
	}

	return mqlFeatures, nil
}

func (wh *mqlWindowsOptionalFeature) id() (string, error) {
	return wh.Name.Data, nil
}

func initWindowsOptionalFeature(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := NewResource(runtime, "windows", nil)
	if err != nil {
		return nil, nil, err
	}
	winResource := obj.(*mqlWindows)

	features := winResource.GetOptionalFeatures()
	if features.Error != nil {
		return nil, nil, features.Error
	}

	for i := range features.Data {
		hf := features.Data[i].(*mqlWindowsOptionalFeature)
		if hf.Name.Data == name {
			return nil, hf, nil
		}
	}

	// if the feature cannot be found we return an error
	return nil, nil, errors.New("could not find feature " + name)
}

func (w *mqlWindows) optionalFeatures() ([]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query features
	encodedCmd := powershell.Encode(windows.QUERY_OPTIONAL_FEATURES)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve optional features: " + string(stderr))
	}

	features, err := windows.ParseWindowsOptionalFeatures(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert features to MQL resource
	mqlFeatures := make([]interface{}, len(features))
	for i, feature := range features {

		mqlFeature, err := CreateResource(w.MqlRuntime, "windows.optionalFeature", map[string]*llx.RawData{
			"name":        llx.StringData(feature.Name),
			"displayName": llx.StringData(feature.DisplayName),
			"description": llx.StringData(feature.Description),
			"enabled":     llx.BoolData(feature.Enabled),
			"state":       llx.IntData(feature.State),
		})
		if err != nil {
			return nil, err
		}

		mqlFeatures[i] = mqlFeature
	}

	return mqlFeatures, nil
}

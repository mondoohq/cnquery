package resources

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/resources/packages"
	"go.mondoo.com/cnquery/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/providers/os/resources/windows"
)

func (s *mqlWindows) id() (string, error) {
	return "windows", nil
}

func (s *mqlWindows) computerInfo() (map[string]interface{}, error) {
	// encode the powershell command
	encodedCmd := powershell.Encode(windows.PSGetComputerInfo)
	out, err := runCommand(s.MqlRuntime, encodedCmd)
	if err != nil {
		return nil, err
	}

	return windows.ParseComputerInfo([]byte(out))
}

func (wh *mqlWindowsHotfix) id() (string, error) {
	return wh.HotfixId.Data, nil
}

func initWindowsHotfix(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
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

	o, err := CreateResource(runtime, "windows", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	winResource := o.(*mqlWindows)

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
	encodedCmd := powershell.Encode(packages.WINDOWS_QUERY_HOTFIXES)
	out, err := runCommand(w.MqlRuntime, encodedCmd)
	if err != nil {
		return nil, err
	}

	hotfixes, err := packages.ParseWindowsHotfixes([]byte(out))
	if err != nil {
		return nil, err
	}

	// convert hotfixes to MQL resource
	mqlHotFixes := make([]interface{}, len(hotfixes))
	for i, hf := range hotfixes {
		var installedOn *llx.RawData
		if time := hf.InstalledOnTime(); time != nil {
			installedOn = llx.TimeData(*time)
		} else {
			installedOn = llx.TimeData(llx.NeverPastTime)
		}

		o, err := CreateResource(w.MqlRuntime, "windows.hotfix", map[string]*llx.RawData{
			"hotfixId":    llx.StringData(hf.HotFixId),
			"caption":     llx.StringData(hf.Caption),
			"description": llx.StringData(hf.Description),
			"installedOn": installedOn,
			"installedBy": llx.StringData(hf.InstalledBy),
		})
		if err != nil {
			return nil, err
		}
		mqlHotFixes[i] = o.(*mqlWindowsHotfix)
	}

	return mqlHotFixes, nil
}

func (wh *mqlWindowsFeature) id() (string, error) {
	return wh.Path.Data, nil
}

func initWindowsFeature(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
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

	o, err := CreateResource(runtime, "windows", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	winResource := o.(*mqlWindows)

	feats := winResource.GetFeatures()
	if feats.Error != nil {
		return nil, nil, feats.Error
	}

	for i := range feats.Data {
		hf := feats.Data[i].(*mqlWindowsFeature)
		if hf.Name.Data == name {
			return nil, hf, nil
		}
	}

	// if the feature cannot be found we return an error
	return nil, nil, errors.New("could not find feature " + name)
}

func (w *mqlWindows) features() ([]interface{}, error) {
	encodedCmd := powershell.Encode(windows.QUERY_FEATURES)
	out, err := runCommand(w.MqlRuntime, encodedCmd)
	if err != nil {
		return nil, err
	}

	features, err := windows.ParseWindowsFeatures([]byte(out))
	if err != nil {
		return nil, err
	}

	// convert features to MQL resource
	mqlFeatures := make([]interface{}, len(features))
	for i, feature := range features {

		o, err := CreateResource(w.MqlRuntime, "windows.feature", map[string]*llx.RawData{
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
		mqlFeatures[i] = o.(*mqlWindowsFeature)
	}

	return mqlFeatures, nil
}

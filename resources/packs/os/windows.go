package os

import (
	"errors"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core/packages"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
	"go.mondoo.io/mondoo/resources/packs/os/windows"
)

func (s *mqlWindows) id() (string, error) {
	return "windows", nil
}

func (s *mqlWindows) GetComputerInfo() (map[string]interface{}, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	cmd := windows.PSGetComputerInfo

	// encode the powershell command
	encodedCmd := powershell.Encode(cmd)
	executedCmd, err := osProvider.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	return windows.ParseComputerInfo(executedCmd.Stdout)
}

func (wh *mqlWindowsHotfix) id() (string, error) {
	return wh.HotfixId()
}

func (p *mqlWindowsHotfix) init(args *resources.Args) (*resources.Args, WindowsHotfix, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	nameRaw := (*args)["hotfixId"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := p.MotorRuntime.CreateResource("windows")
	if err != nil {
		return nil, nil, err
	}
	winResource := obj.(Windows)

	hotfixes, err := winResource.Hotfixes()
	if err != nil {
		return nil, nil, err
	}

	for i := range hotfixes {
		hf := hotfixes[i].(WindowsHotfix)
		id, err := hf.HotfixId()
		if err == nil && id == name {
			return nil, hf, nil
		}
	}

	// if the hotfix cannot be found we return an error
	return nil, nil, errors.New("could not find hotfix " + name)
}

func (w *mqlWindows) GetHotfixes() ([]interface{}, error) {
	osProvider, err := osProvider(w.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// query hotfixes
	encodedCmd := powershell.Encode(packages.WINDOWS_QUERY_HOTFIXES)
	executedCmd, err := osProvider.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	hotfixes, err := packages.ParseWindowsHotfixes(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert hotfixes to MQL resource
	mqlHotFixes := make([]interface{}, len(hotfixes))
	for i, hf := range hotfixes {

		mqlHotfix, err := w.MotorRuntime.CreateResource("windows.hotfix",
			"hotfixId", hf.HotFixId,
			"caption", hf.Caption,
			"description", hf.Description,
			"installedOn", hf.InstalledOnTime(),
			"installedBy", hf.InstalledBy,
		)
		if err != nil {
			return nil, err
		}

		mqlHotFixes[i] = mqlHotfix.(WindowsHotfix)
	}

	return mqlHotFixes, nil
}

func (wh *mqlWindowsFeature) id() (string, error) {
	return wh.Path()
}

func (p *mqlWindowsFeature) init(args *resources.Args) (*resources.Args, WindowsFeature, error) {
	if len(*args) > 2 {
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

	obj, err := p.MotorRuntime.CreateResource("windows")
	if err != nil {
		return nil, nil, err
	}
	winResource := obj.(Windows)

	features, err := winResource.Features()
	if err != nil {
		return nil, nil, err
	}

	for i := range features {
		hf := features[i].(WindowsFeature)
		id, err := hf.Name()
		if err == nil && id == name {
			return nil, hf, nil
		}
	}

	// if the feature cannot be found we return an error
	return nil, nil, errors.New("could not find feature " + name)
}

func (w *mqlWindows) GetFeatures() ([]interface{}, error) {
	osProvider, err := osProvider(w.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// query features
	encodedCmd := powershell.Encode(windows.QUERY_FEATURES)
	executedCmd, err := osProvider.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	features, err := windows.ParseWindowsFeatures(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert features to MQL resource
	mqlFeatures := make([]interface{}, len(features))
	for i, feature := range features {

		mqlFeature, err := w.MotorRuntime.CreateResource("windows.feature",
			"path", feature.Path,
			"name", feature.Name,
			"displayName", feature.DisplayName,
			"description", feature.Description,
			"installed", feature.Installed,
			"installState", feature.InstallState,
		)
		if err != nil {
			return nil, err
		}

		mqlFeatures[i] = mqlFeature.(WindowsFeature)
	}

	return mqlFeatures, nil
}

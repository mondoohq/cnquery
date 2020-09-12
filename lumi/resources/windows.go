package resources

import (
	"errors"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (s *lumiWindows) id() (string, error) {
	return "windows", nil
}

func (s *lumiWindows) GetComputerInfo() (map[string]interface{}, error) {
	cmd := windows.PSGetComputerInfo

	// encode the powershell command
	encodedCmd := powershell.Encode(cmd)
	executedCmd, err := s.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	return windows.ParseComputerInfo(executedCmd.Stdout)
}

func (wh *lumiWindowsHotfix) id() (string, error) {
	return wh.HotfixId()
}

func (p *lumiWindowsHotfix) init(args *lumi.Args) (*lumi.Args, WindowsHotfix, error) {
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

	obj, err := p.Runtime.CreateResource("windows")
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

func (w *lumiWindows) GetHotfixes() ([]interface{}, error) {
	// query hotfixes
	encodedCmd := powershell.Encode(packages.WINDOWS_QUERY_HOTFIXES)
	executedCmd, err := w.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	hotfixes, err := packages.ParseWindowsHotfixes(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert hotfixes to lumi resource
	lumiHotFixes := make([]interface{}, len(hotfixes))
	for i, hf := range hotfixes {

		lumiHotfix, err := w.Runtime.CreateResource("windows.hotfix",
			"hotfixId", hf.HotFixId,
			"caption", hf.Caption,
			"description", hf.Description,
			"installedOn", hf.InstalledOnTime(),
			"installedBy", hf.InstalledBy,
		)
		if err != nil {
			return nil, err
		}

		lumiHotFixes[i] = lumiHotfix.(WindowsHotfix)
	}

	return lumiHotFixes, nil
}

func (wh *lumiWindowsFeature) id() (string, error) {
	return wh.Path()
}

func (p *lumiWindowsFeature) init(args *lumi.Args) (*lumi.Args, WindowsFeature, error) {
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

	obj, err := p.Runtime.CreateResource("windows")
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

func (w *lumiWindows) GetFeatures() ([]interface{}, error) {
	// query features
	encodedCmd := powershell.Encode(windows.QUERY_FEATURES)
	executedCmd, err := w.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	features, err := windows.ParseWindowsFeatures(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert features to lumi resource
	lumiFeatures := make([]interface{}, len(features))
	for i, feature := range features {

		lumiFeature, err := w.Runtime.CreateResource("windows.feature",
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

		lumiFeatures[i] = lumiFeature.(WindowsFeature)
	}

	return lumiFeatures, nil
}

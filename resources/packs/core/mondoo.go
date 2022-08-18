package core

import (
	"errors"
	"sort"

	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/cli/execruntime"
)

func (m *lumiMondoo) id() (string, error) {
	return "mondoo", nil
}

func (m *lumiMondoo) GetVersion() (string, error) {
	return mondoo.GetVersion(), nil
}

func (m *lumiMondoo) GetBuild() (string, error) {
	return mondoo.GetBuild(), nil
}

func (m *lumiMondoo) GetNulllist() ([]interface{}, error) {
	return nil, nil
}

func (m *lumiMondoo) GetResources() ([]interface{}, error) {
	n := m.MotorRuntime.Registry.Names()
	sort.Strings(n)
	res := make([]interface{}, len(n))
	for i, s := range n {
		res[i] = s
	}
	return res, nil
}

type runtimeEnv struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (m *lumiMondoo) GetJobEnvironment() (map[string]interface{}, error) {
	// get the local agent runtime information
	ciEnv := execruntime.Detect()

	re := &runtimeEnv{
		ID:   ciEnv.Namespace,
		Name: ciEnv.Name,
	}

	return JsonToDict(re)
}

func (m *lumiMondoo) GetCapabilities() ([]interface{}, error) {
	capabilities := []interface{}{}
	caps := m.MotorRuntime.Motor.Provider.Capabilities()
	for i := range caps {
		capabilities = append(capabilities, caps[i].String())
	}
	return capabilities, nil
}

func (m *lumiMondooAsset) id() (string, error) {
	return "mondoo.asset", nil
}

func (m *lumiMondooAsset) GetPlatformIDs() ([]interface{}, error) {
	asset := m.MotorRuntime.Motor.GetAsset()
	if asset == nil {
		return nil, errors.New("unimplemented")
	}
	return StrSliceToInterface(asset.PlatformIds), nil
}

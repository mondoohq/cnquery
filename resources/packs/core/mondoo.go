package core

import (
	"errors"
	"sort"

	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.io/mondoo"
)

func (m *mqlMondoo) id() (string, error) {
	return "mondoo", nil
}

func (m *mqlMondoo) GetVersion() (string, error) {
	return mondoo.GetVersion(), nil
}

func (m *mqlMondoo) GetBuild() (string, error) {
	return mondoo.GetBuild(), nil
}

func (m *mqlMondoo) GetNulllist() ([]interface{}, error) {
	return nil, nil
}

func (m *mqlMondoo) GetResources() ([]interface{}, error) {
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

func (m *mqlMondoo) GetJobEnvironment() (map[string]interface{}, error) {
	// get the local agent runtime information
	ciEnv := execruntime.Detect()

	re := &runtimeEnv{
		ID:   ciEnv.Namespace,
		Name: ciEnv.Name,
	}

	return JsonToDict(re)
}

func (m *mqlMondoo) GetCapabilities() ([]interface{}, error) {
	capabilities := []interface{}{}
	caps := m.MotorRuntime.Motor.Provider.Capabilities()
	for i := range caps {
		capabilities = append(capabilities, caps[i].String())
	}
	return capabilities, nil
}

func (m *mqlMondooAsset) id() (string, error) {
	return "mondoo.asset", nil
}

func (m *mqlMondooAsset) GetPlatformIDs() ([]interface{}, error) {
	asset := m.MotorRuntime.Motor.GetAsset()
	if asset == nil {
		return nil, errors.New("unimplemented")
	}
	return StrSliceToInterface(asset.PlatformIds), nil
}

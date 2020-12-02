package resources

import (
	"sort"

	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/cli/execruntime"
)

func (m *lumiMondoo) id() (string, error) {
	return "", nil
}

func (m *lumiMondoo) GetVersion() (string, error) {
	return mondoo.GetVersion(), nil
}

func (m *lumiMondoo) GetBuild() (string, error) {
	return mondoo.GetBuild(), nil
}

func (m *lumiMondoo) GetResources() ([]interface{}, error) {
	n := m.Runtime.Registry.Names()
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

	return jsonToDict(re)
}

func (m *lumiMondoo) GetCapabilities() ([]interface{}, error) {
	capabilities := []interface{}{}
	caps := m.Runtime.Motor.Transport.Capabilities()
	for i := range caps {
		capabilities = append(capabilities, caps[i].String())
	}
	return capabilities, nil
}

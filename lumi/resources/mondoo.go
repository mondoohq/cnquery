package resources

import (
	"sort"

	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/cli/execruntime"
)

func (s *lumiMondoo) id() (string, error) {
	return "", nil
}

func (s *lumiMondoo) GetVersion() (string, error) {
	return mondoo.GetVersion(), nil
}

func (s *lumiMondoo) GetBuild() (string, error) {
	return mondoo.GetBuild(), nil
}

func (s *lumiMondoo) GetResources() ([]interface{}, error) {
	n := s.Runtime.Registry.Names()
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

func (s *lumiMondoo) GetJobEnvironment() (map[string]interface{}, error) {
	// get the local agent runtime information
	ciEnv := execruntime.Detect()

	re := &runtimeEnv{
		ID:   ciEnv.Namespace,
		Name: ciEnv.Name,
	}

	return jsonToDict(re)
}

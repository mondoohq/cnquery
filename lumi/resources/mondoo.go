package resources

import (
	"sort"

	"go.mondoo.io/mondoo"
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

package os

import (
	"errors"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/os/windows"
)

func (s *mqlSecpol) id() (string, error) {
	return "secpol", nil
}

func (s *mqlSecpol) policy() (*windows.Secpol, error) {
	var policy *windows.Secpol
	data, ok := s.Cache.Load("secpol")
	if ok {
		policy, ok := data.Data.(*windows.Secpol)
		if ok {
			return policy, nil
		}
	}

	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	encoded := powershell.Encode(windows.SecpolScript)

	cmd, err := osProvider.RunCommand(encoded)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not run secpol script"))
	}

	policy, err = windows.ParseSecpol(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	s.Cache.Store("secpol", &resources.CacheEntry{Data: policy})
	return policy, nil
}

func (s *mqlSecpol) GetSystemaccess() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.SystemAccess, nil
}

func (s *mqlSecpol) GetEventaudit() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.EventAudit, nil
}

func (s *mqlSecpol) GetRegistryvalues() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.RegistryValues, nil
}

func (s *mqlSecpol) GetPrivilegerights() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.PrivilegeRights, nil
}

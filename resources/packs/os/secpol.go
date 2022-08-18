package os

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
	"go.mondoo.io/mondoo/resources/packs/os/windows"
)

func (s *lumiSecpol) id() (string, error) {
	return "secpol", nil
}

func (s *lumiSecpol) policy() (*windows.Secpol, error) {
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
		return nil, errors.Wrap(err, "could not run secpol script")
	}

	policy, err = windows.ParseSecpol(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	s.Cache.Store("secpol", &lumi.CacheEntry{Data: policy})
	return policy, nil
}

func (s *lumiSecpol) GetSystemaccess() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.SystemAccess, nil
}

func (s *lumiSecpol) GetEventaudit() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.EventAudit, nil
}

func (s *lumiSecpol) GetRegistryvalues() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.RegistryValues, nil
}

func (s *lumiSecpol) GetPrivilegerights() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.PrivilegeRights, nil
}

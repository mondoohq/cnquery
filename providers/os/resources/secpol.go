// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

type mqlSecpolInternal struct {
	_policy *windows.Secpol
}

func (s *mqlSecpol) policy() (*windows.Secpol, error) {
	if s._policy != nil {
		return s._policy, nil
	}

	out, err := s.runPowershell(windows.SecpolScript)
	if err != nil {
		return nil, fmt.Errorf("could not run secedit: %w", err)
	}

	policy, err := windows.ParseSecpol(strings.NewReader(out), s.resolveSids)
	if err != nil {
		return nil, err
	}
	s._policy = policy

	return policy, nil
}

func (s *mqlSecpol) resolveSids(names []string) (map[string]string, error) {
	out, err := s.runPowershell(windows.SidLookupScript(names))
	if err != nil {
		return nil, err
	}
	return windows.ParseSidLookup(strings.NewReader(out))
}

func (s *mqlSecpol) runPowershell(script string) (string, error) {
	o, err := CreateResource(s.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(powershell.Encode(script)),
	})
	if err != nil {
		return "", err
	}

	out := o.(*mqlCommand).GetStdout()
	if out.Error != nil {
		return "", out.Error
	}
	return out.Data, nil
}

func (s *mqlSecpol) systemaccess() (map[string]any, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.SystemAccess, nil
}

func (s *mqlSecpol) eventaudit() (map[string]any, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.EventAudit, nil
}

func (s *mqlSecpol) registryvalues() (map[string]any, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.RegistryValues, nil
}

func (s *mqlSecpol) privilegerights() (map[string]any, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.PrivilegeRights, nil
}

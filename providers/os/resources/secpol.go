// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/v10/providers/os/resources/windows"
)

type mqlSecpolInternal struct {
	_policy *windows.Secpol
}

func (s *mqlSecpol) policy() (*windows.Secpol, error) {
	if s._policy != nil {
		return s._policy, nil
	}

	encoded := powershell.Encode(windows.SecpolScript)

	o, err := CreateResource(s.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(encoded),
	})
	if err != nil {
		return nil, err
	}

	cmd := o.(*mqlCommand)
	out := cmd.GetStdout()
	if out.Error != nil {
		return nil, fmt.Errorf("could not run auditpol: " + out.Error.Error())
	}

	policy, err := windows.ParseSecpol(strings.NewReader(out.Data))
	if err != nil {
		return nil, err
	}
	s._policy = policy

	return policy, nil
}

func (s *mqlSecpol) systemaccess() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.SystemAccess, nil
}

func (s *mqlSecpol) eventaudit() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.EventAudit, nil
}

func (s *mqlSecpol) registryvalues() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.RegistryValues, nil
}

func (s *mqlSecpol) privilegerights() (map[string]interface{}, error) {
	policy, err := s.policy()
	if err != nil {
		return nil, err
	}
	return policy.PrivilegeRights, nil
}

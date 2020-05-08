package groups

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/lumi/resources/powershell"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

type WindowsSID struct {
	BinaryLength     int
	AccountDomainSid *string
	Value            string
}

type WindowsLocalGroup struct {
	Name            string
	Description     string
	PrincipalSource int
	SID             WindowsSID
	ObjectClass     string
}

func ParseWindowsLocalGroups(r io.Reader) ([]WindowsLocalGroup, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var localGroups []WindowsLocalGroup
	err = json.Unmarshal(data, &localGroups)
	if err != nil {
		return nil, err
	}

	return localGroups, nil
}

type WindowsGroupManager struct {
	motor *motor.Motor
}

func (s *WindowsGroupManager) Name() string {
	return "Windows Group Manager"
}

func (s *WindowsGroupManager) Group(id string) (*Group, error) {
	groups, err := s.List()
	if err != nil {
		return nil, err
	}

	return findGroup(groups, id)
}

func (s *WindowsGroupManager) List() ([]*Group, error) {
	powershellCmd := "Get-LocalGroup | ConvertTo-Json"
	c, err := s.motor.Transport.RunCommand(powershell.Wrap(powershellCmd))
	if err != nil {
		return nil, err
	}
	winUsers, err := ParseWindowsLocalGroups(c.Stdout)

	res := []*Group{}
	for i := range winUsers {
		res = append(res, winToGroup(winUsers[i]))
	}
	return res, nil
}

func winToGroup(g WindowsLocalGroup) *Group {
	return &Group{
		ID:      g.SID.Value,
		Sid:     g.SID.Value,
		Gid:     -1, // TODO: not its suboptimal, but lets make sure to avoid runtime conflicts for now
		Name:    g.Name,
		Members: []string{},
	}
}

package users

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor/providers/os"
)

type WindowsSID struct {
	BinaryLength     int
	AccountDomainSid *string
	Value            string
}

// NOTE: there is some overlap with windows groups
type WindowsLocalUser struct {
	// same values as in group
	Name            string
	Description     string
	PrincipalSource int
	SID             WindowsSID
	ObjectClass     string

	// special attributes for user
	Enabled                bool
	FullName               string
	PasswordRequired       bool
	UserMayChangePassword  bool
	AccountExpires         *string
	PasswordChangeableDate *string
	PasswordExpires        *string
	PasswordLastSet        *string
	LastLogon              *string
}

func ParseWindowsLocalUsers(r io.Reader) ([]WindowsLocalUser, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var localUsers []WindowsLocalUser
	err = json.Unmarshal(data, &localUsers)
	if err != nil {
		return nil, err
	}

	return localUsers, nil
}

type WindowsUserManager struct {
	provider os.OperatingSystemProvider
}

func (s *WindowsUserManager) Name() string {
	return "Windows User Manager"
}

func (s *WindowsUserManager) User(id string) (*User, error) {
	users, err := s.List()
	if err != nil {
		return nil, err
	}

	return findUser(users, id)
}

func (s *WindowsUserManager) List() ([]*User, error) {
	powershellCmd := "Get-LocalUser | ConvertTo-Json"
	c, err := s.provider.RunCommand(powershell.Wrap(powershellCmd))
	if err != nil {
		return nil, err
	}
	winUsers, err := ParseWindowsLocalUsers(c.Stdout)
	if err != nil {
		return nil, err
	}

	res := []*User{}
	for i := range winUsers {
		res = append(res, winToUser(winUsers[i]))
	}
	return res, nil
}

func winToUser(g WindowsLocalUser) *User {
	// TODO: consider to store additional attributes in key-value pairs
	return &User{
		ID:      g.SID.Value,
		Sid:     g.SID.Value,
		Uid:     -1, // TODO: not its suboptimal, but lets make sure to avoid runtime conflicts for now
		Gid:     -1,
		Name:    g.Name,
		Enabled: g.Enabled,
	}
}

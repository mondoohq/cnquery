// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package users

import (
	"encoding/json"
	"io"
	"runtime"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/powershell"
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
	data, err := io.ReadAll(r)
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
	conn shared.Connection
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
	// If we are running locally on Windows, use native API for better performance
	// (~1-10ms vs ~200-500ms for PowerShell)
	if s.conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		log.Debug().Msg("using native Windows API for user enumeration")
		users, err := GetNativeUsers()
		if err == nil {
			return users, nil
		}
		// Fall back to PowerShell if native API fails
		log.Debug().Err(err).Msg("native Windows API failed, falling back to PowerShell")
	}

	powershellCmd := "Get-LocalUser | ConvertTo-Json"
	c, err := s.conn.RunCommand(powershell.Wrap(powershellCmd))
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

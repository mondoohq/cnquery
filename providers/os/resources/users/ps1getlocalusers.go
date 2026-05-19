// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package users

import (
	"encoding/json"
	"io"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
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

	// On-disk profile path, joined from HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList.
	// Empty for accounts that have never logged in (Guest, DefaultAccount, WDAGUtilityAccount, ...).
	LocalPath string
}

// getLocalUsersScript enumerates local users and joins each with their on-disk
// profile path. Profile paths come from the ProfileList registry hive — the same
// data Win32_UserProfile exposes — to avoid WMI latency on hosts with many
// cached profiles (RDS, Citrix). The output is a JSON array shaped like the
// previous `Get-LocalUser | ConvertTo-Json` output, with an extra LocalPath field.
const getLocalUsersScript = `
$profiles = @{}
Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList' -ErrorAction SilentlyContinue |
    ForEach-Object {
        $sid = Split-Path $_.Name -Leaf
        $path = (Get-ItemProperty -LiteralPath $_.PSPath -ErrorAction SilentlyContinue).ProfileImagePath
        if ($path) { $profiles[$sid] = $path }
    }
$out = foreach ($u in Get-LocalUser) {
    $u | Add-Member -NotePropertyName LocalPath -NotePropertyValue $profiles[$u.SID.Value] -PassThru -Force
}
ConvertTo-Json -InputObject @($out) -Depth 4
`

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
	c, err := s.conn.RunCommand(powershell.Encode(getLocalUsersScript))
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
		Home:    g.LocalPath,
		Enabled: g.Enabled,
	}
}

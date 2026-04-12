// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "fmt"

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

type UserInfo struct {
	UserID    string `json:"userid"`
	Email     string `json:"email"`
	Enable    int    `json:"enable"`
	Expire    int64  `json:"expire"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Groups    string `json:"groups"` // comma-separated
	RealmType string `json:"realm-type"`
	Tokens    []struct {
		TokenID string `json:"tokenid"`
		Comment string `json:"comment"`
		Expire  int64  `json:"expire"`
		Privsep int    `json:"privsep"`
	} `json:"tokens"`
}

func (c *PveConnection) GetUsers() ([]UserInfo, error) {
	var users []UserInfo
	if err := c.apiGet("/access/users?full=1", &users); err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	return users, nil
}

// ---------------------------------------------------------------------------
// User tokens
// ---------------------------------------------------------------------------

type TokenInfo struct {
	TokenID string `json:"tokenid"`
	Comment string `json:"comment"`
	Expire  int64  `json:"expire"`
	Privsep int    `json:"privsep"`
}

func (c *PveConnection) GetUserTokens(userid string) ([]TokenInfo, error) {
	var tokens []TokenInfo
	path := fmt.Sprintf("/access/users/%s/token", userid)
	if err := c.apiGet(path, &tokens); err != nil {
		return nil, fmt.Errorf("failed to get tokens for user %s: %w", userid, err)
	}
	return tokens, nil
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

type RoleInfo struct {
	RoleID  string `json:"roleid"`
	Privs   string `json:"privs"` // comma-separated
	Special int    `json:"special"`
}

func (c *PveConnection) GetRoles() ([]RoleInfo, error) {
	var roles []RoleInfo
	if err := c.apiGet("/access/roles", &roles); err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	return roles, nil
}

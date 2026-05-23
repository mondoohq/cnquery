// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

// init functions populate a resource when it was looked up by id only —
// e.g. an ACL entry resolving its target user, group, role, or token.

func initProxmoxUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	userID := args["id"].Value.(string)
	if userID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	users, err := conn.GetUsers()
	if err != nil {
		return nil, nil, err
	}
	for _, u := range users {
		if u.UserID != userID {
			continue
		}
		var groups []any
		if u.Groups != "" {
			for _, g := range strings.Split(u.Groups, ",") {
				groups = append(groups, g)
			}
		}
		realm := ""
		if parts := strings.SplitN(u.UserID, "@", 2); len(parts) == 2 {
			realm = parts[1]
		}
		args["email"] = llx.StringData(u.Email)
		args["enable"] = llx.BoolData(u.Enable == 1)
		args["expire"] = llx.IntData(u.Expire)
		args["firstname"] = llx.StringData(u.Firstname)
		args["lastname"] = llx.StringData(u.Lastname)
		args["groups"] = llx.ArrayData(groups, "\x02")
		args["realm"] = llx.StringData(realm)
		args["realmType"] = llx.StringData(u.RealmType)
		args["tfaLockedUntil"] = llx.IntData(u.TFALockedUntil)
		return args, nil, nil
	}
	// User referenced by ACL no longer exists — return the bare resource
	// so audits can still see the dangling reference rather than erroring.
	return args, nil, nil
}

func initProxmoxGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	groupID := args["id"].Value.(string)
	if groupID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	groups, err := conn.GetGroups()
	if err != nil {
		return nil, nil, err
	}
	for _, g := range groups {
		if g.GroupID != groupID {
			continue
		}
		var memberIds []any
		if g.Users != "" {
			for _, u := range strings.Split(g.Users, ",") {
				if u = strings.TrimSpace(u); u != "" {
					memberIds = append(memberIds, u)
				}
			}
		}
		args["comment"] = llx.StringData(g.Comment)
		args["memberIds"] = llx.ArrayData(memberIds, "\x02")
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	roleID := args["id"].Value.(string)
	if roleID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	roles, err := conn.GetRoles()
	if err != nil {
		return nil, nil, err
	}
	for _, r := range roles {
		if r.RoleID != roleID {
			continue
		}
		var privs []any
		if r.Privs != "" {
			for _, p := range strings.Split(r.Privs, ",") {
				privs = append(privs, p)
			}
		}
		args["privs"] = llx.ArrayData(privs, "\x02")
		args["special"] = llx.BoolData(r.Special == 1)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxStorage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	storageID := args["id"].Value.(string)
	if storageID == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.PveConnection)
	storages, err := conn.GetStorages()
	if err != nil {
		return nil, nil, err
	}
	for _, s := range storages {
		if s.Storage != storageID {
			continue
		}
		var usagePct float64
		if s.UsedFrac > 0 {
			usagePct = s.UsedFrac * 100.0
		} else if s.Total > 0 {
			usagePct = float64(s.Used) / float64(s.Total) * 100.0
		}
		args["type"] = llx.StringData(s.Type)
		args["content"] = llx.StringData(s.Content)
		args["path"] = llx.StringData(s.Path)
		args["enabled"] = llx.BoolData(s.Enabled != 0)
		args["shared"] = llx.BoolData(s.Shared != 0)
		args["total"] = llx.IntData(s.Total)
		args["used"] = llx.IntData(s.Used)
		args["available"] = llx.IntData(s.Avail)
		args["usagePercent"] = llx.FloatData(usagePct)
		return args, nil, nil
	}
	return args, nil, nil
}

func initProxmoxToken(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 || args["id"] == nil {
		return args, nil, nil
	}
	tokenID := args["id"].Value.(string)
	if tokenID == "" {
		return args, nil, nil
	}
	// Token IDs are user@realm!tokenid — derive the owner to call the
	// per-user endpoint instead of paging through every user.
	bangIdx := strings.LastIndex(tokenID, "!")
	if bangIdx <= 0 {
		return args, nil, nil
	}
	owner := tokenID[:bangIdx]
	leaf := tokenID[bangIdx+1:]
	conn := runtime.Connection.(*connection.PveConnection)
	tokens, err := conn.GetUserTokens(owner)
	if err != nil {
		return args, nil, nil
	}
	for _, t := range tokens {
		if t.TokenID != leaf {
			continue
		}
		args["comment"] = llx.StringData(t.Comment)
		args["expire"] = llx.IntData(t.Expire)
		args["privsep"] = llx.BoolData(t.Privsep == 1)
		return args, nil, nil
	}
	return args, nil, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/iru/connection"
	"go.mondoo.com/mql/v13/providers/iru/connection/client"
)

func (r *mqlIru) users() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.IruConnection)
	users, err := conn.ListUsers()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(users))
	for _, u := range users {
		item, err := CreateResource(r.MqlRuntime, "iru.user", userArgs(&u))
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}

func (u *mqlIruUser) id() (string, error) {
	return "iru.user/" + u.Id.Data, nil
}

// initIruUser hydrates an iru.user created by id (e.g. via
// `iru.user(id: "...")` or as a cross-reference from iru.device.user)
// using the tenant-wide user listing memoized on the connection.
func initIruUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Value == nil {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.IruConnection)
	users, err := conn.ListUsers()
	if err != nil {
		return args, nil, err
	}
	for i := range users {
		if users[i].ID != id {
			continue
		}
		return userArgs(&users[i]), nil, nil
	}
	return nil, nil, fmt.Errorf("iru.user with id %q not found", id)
}

func userArgs(u *client.User) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"id":         llx.StringData(u.ID),
		"name":       llx.StringData(u.Name),
		"email":      llx.StringData(u.Email),
		"username":   llx.StringData(u.Username),
		"department": llx.StringData(u.Department),
		"jobTitle":   llx.StringData(u.JobTitle),
		"archived":   llx.BoolData(u.IsArchived),
	}
}

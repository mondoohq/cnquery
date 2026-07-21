// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

// initOktaUser allows callers to construct an okta.user via NewResource by id.
// When only an id is provided, this fetches the user lazily (cached by the
// runtime) so referencing the same user across resources does not N+1 the API.
func initOktaUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// If we already have the full set of fields, no fetch needed.
	if len(args) > 1 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Value == nil {
		// Bare resource construction (no id) is a valid empty state.
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()
	user, _, err := client.UserAPI.GetUser(ctx, id).Execute()
	if err != nil {
		return nil, nil, err
	}

	userArgs, err := oktaUserArgs(user)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range userArgs {
		args[k] = v
	}
	return args, nil, nil
}

func (o *mqlOkta) users() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	userSetSlice, resp, err := client.UserAPI.ListUsers(ctx).Limit(queryLimit).Execute()
	if err != nil {
		return nil, err
	}

	if len(userSetSlice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []okta.User) error {
		for i := range datalist {
			r, err := newMqlOktaUser(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(userSetSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var userSetSlice []okta.User
		resp, err = resp.Next(&userSetSlice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(userSetSlice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

// oktaUserArgs builds the okta.user resource fields from any of the v5 SDK's
// user-shaped types (User, UserGetSingleton, GroupMember). They share the same
// JSON shape, so we normalize through JSON into a single okta.User before
// mapping — this keeps one code path for every place a user is materialized.
func oktaUserArgs(src any) (map[string]*llx.RawData, error) {
	var user okta.User
	switch v := src.(type) {
	case *okta.User:
		user = *v
	default:
		raw, err := json.Marshal(src)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &user); err != nil {
			return nil, err
		}
	}

	userType, err := convert.JsonToDict(user.Type)
	if err != nil {
		return nil, err
	}
	var userTypeId string
	if user.Type != nil {
		userTypeId = oktaStr(user.Type.Id)
	}
	credentials, err := convert.JsonToDict(user.Credentials)
	if err != nil {
		return nil, err
	}
	profileDict, err := convert.JsonToDict(user.Profile)
	if err != nil {
		return nil, err
	}

	return map[string]*llx.RawData{
		"id":                    llx.StringData(oktaStr(user.Id)),
		"type":                  llx.DictData(userType),
		"typeId":                llx.StringData(userTypeId),
		"credentials":           llx.DictData(credentials),
		"activated":             llx.TimeDataPtr(user.Activated.Get()),
		"created":               llx.TimeDataPtr(user.Created),
		"lastLogin":             llx.TimeDataPtr(user.LastLogin.Get()),
		"lastUpdated":           llx.TimeDataPtr(user.LastUpdated),
		"passwordChanged":       llx.TimeDataPtr(user.PasswordChanged.Get()),
		"profile":               llx.DictData(profileDict),
		"status":                llx.StringData(oktaStr(user.Status)),
		"statusChanged":         llx.TimeDataPtr(user.StatusChanged.Get()),
		"transitioningToStatus": llx.StringData(oktaStr(user.TransitioningToStatus.Get())),
	}, nil
}

func newMqlOktaUser(runtime *plugin.Runtime, src any) (*mqlOktaUser, error) {
	args, err := oktaUserArgs(src)
	if err != nil {
		return nil, err
	}
	r, err := CreateResource(runtime, "okta.user", args)
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaUser), nil
}

func (o *mqlOktaUser) id() (string, error) {
	return "okta.user/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaUser) roles() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	roles, resp, err := client.RoleAssignmentAPI.ListAssignedRolesForUser(ctx, o.Id.Data).Execute()
	if err != nil {
		return nil, err
	}
	res := []any{}

	appendEntry := func(datalist []okta.Role) error {
		for i := range datalist {
			mqlOktaRole, err := newMqlOktaRole(o.MqlRuntime, &datalist[i], "user", o.Id.Data)
			if err != nil {
				return err
			}
			res = append(res, mqlOktaRole)
		}
		return nil
	}
	err = appendEntry(roles)
	if err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var userRoles []okta.Role
		resp, err = resp.Next(&userRoles)
		if err != nil {
			return nil, err
		}
		err = appendEntry(userRoles)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

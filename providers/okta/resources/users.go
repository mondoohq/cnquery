// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
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
	user, _, err := client.User.GetUser(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	userType, err := convert.JsonToDict(user.Type)
	if err != nil {
		return nil, nil, err
	}
	var userTypeId string
	if user.Type != nil {
		userTypeId = user.Type.Id
	}
	credentials, err := convert.JsonToDict(user.Credentials)
	if err != nil {
		return nil, nil, err
	}
	profileDict := map[string]any{}
	if user.Profile != nil {
		for k, v := range *user.Profile {
			profileDict[k] = v
		}
	}

	args["id"] = llx.StringData(user.Id)
	args["type"] = llx.DictData(userType)
	args["typeId"] = llx.StringData(userTypeId)
	args["credentials"] = llx.DictData(credentials)
	args["activated"] = llx.TimeDataPtr(user.Activated)
	args["created"] = llx.TimeDataPtr(user.Created)
	args["lastLogin"] = llx.TimeDataPtr(user.LastLogin)
	args["lastUpdated"] = llx.TimeDataPtr(user.LastUpdated)
	args["passwordChanged"] = llx.TimeDataPtr(user.PasswordChanged)
	args["profile"] = llx.DictData(profileDict)
	args["status"] = llx.StringData(user.Status)
	args["statusChanged"] = llx.TimeDataPtr(user.StatusChanged)
	args["transitioningToStatus"] = llx.StringData(user.TransitioningToStatus)
	return args, nil, nil
}

func (o *mqlOkta) users() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	userSetSlice, resp, err := client.User.ListUsers(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(userSetSlice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []*okta.User) error {
		for i := range datalist {
			user := datalist[i]
			r, err := newMqlOktaUser(o.MqlRuntime, user)
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
		var userSetSlice []*okta.User
		resp, err = resp.Next(ctx, &userSetSlice)
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

func newMqlOktaUser(runtime *plugin.Runtime, user *okta.User) (*mqlOktaUser, error) {
	// FUTURE: change this to actually fetch the whole type and put it in the dict
	userType, err := convert.JsonToDict(user.Type)
	if err != nil {
		return nil, err
	}
	var userTypeId string
	if user.Type != nil {
		userTypeId = user.Type.Id
	}
	credentials, err := convert.JsonToDict(user.Credentials)
	if err != nil {
		return nil, err
	}

	profileDict := map[string]any{}
	if user.Profile != nil {
		for k, v := range *user.Profile {
			profileDict[k] = v
		}
	}
	r, err := CreateResource(runtime, "okta.user", map[string]*llx.RawData{
		"id":                    llx.StringData(user.Id),
		"type":                  llx.DictData(userType),
		"typeId":                llx.StringData(userTypeId),
		"credentials":           llx.DictData(credentials),
		"activated":             llx.TimeDataPtr(user.Activated),
		"created":               llx.TimeDataPtr(user.Created),
		"lastLogin":             llx.TimeDataPtr(user.LastLogin),
		"lastUpdated":           llx.TimeDataPtr(user.LastUpdated),
		"passwordChanged":       llx.TimeDataPtr(user.PasswordChanged),
		"profile":               llx.DictData(profileDict),
		"status":                llx.StringData(user.Status),
		"statusChanged":         llx.TimeDataPtr(user.StatusChanged),
		"transitioningToStatus": llx.StringData(user.TransitioningToStatus),
	})
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
	roles, resp, err := client.User.ListAssignedRolesForUser(ctx, o.Id.Data, query.NewQueryParams(query.WithLimit(queryLimit)))
	if err != nil {
		return nil, err
	}
	res := []any{}

	appendEntry := func(datalist []*okta.Role) error {
		for _, r := range datalist {
			mqlOktaRole, err := newMqlOktaRole(o.MqlRuntime, r)
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
		var userRoles []*okta.Role
		resp, err = resp.Next(ctx, &userRoles)
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

func newMqlOktaRole(runtime *plugin.Runtime, role *okta.Role) (*mqlOktaRole, error) {
	r, err := CreateResource(runtime, "okta.role", map[string]*llx.RawData{
		"id":             llx.StringData(role.Id),
		"assignmentType": llx.StringData(role.AssignmentType),
		"created":        llx.TimeDataPtr(role.Created),
		"lastUpdated":    llx.TimeDataPtr(role.LastUpdated),
		"label":          llx.StringData(role.Label),
		"status":         llx.StringData(role.Status),
		"type":           llx.StringData(role.Type),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaRole), nil
}

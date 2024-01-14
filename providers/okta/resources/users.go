// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/okta/connection"
)

func (o *mqlOkta) users() ([]interface{}, error) {
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

	list := []interface{}{}
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

	profileDict := map[string]interface{}{}
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

func (o *mqlOktaUser) roles() ([]interface{}, error) {
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
	res := []interface{}{}

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

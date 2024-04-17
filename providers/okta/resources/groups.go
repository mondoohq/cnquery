// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/okta/connection"
)

func (o *mqlOkta) groups() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.Group.ListGroups(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.Group) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaGroup(o.MqlRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}

		return nil
	}

	err = appendEntry(slice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var slice []*okta.Group
		resp, err = resp.Next(ctx, &slice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(slice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaGroup(runtime *plugin.Runtime, entry *okta.Group) (interface{}, error) {
	profile, err := convert.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.group", map[string]*llx.RawData{
		"id":                    llx.StringData(entry.Id),
		"name":                  llx.StringData(entry.Profile.Name),
		"description":           llx.StringData(entry.Profile.Description),
		"type":                  llx.StringData(entry.Type),
		"created":               llx.TimeDataPtr(entry.Created),
		"lastMembershipUpdated": llx.TimeDataPtr(entry.LastMembershipUpdated),
		"lastUpdated":           llx.TimeDataPtr(entry.LastUpdated),
		"profile":               llx.DictData(profile),
	})
}

func (o *mqlOktaGroup) id() (string, error) {
	return "okta.group/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaGroup) members() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	groupID := o.Id.Data
	slice, resp, err := client.Group.ListGroupUsers(ctx, groupID, query.NewQueryParams(query.WithLimit(queryLimit)))

	if len(slice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.User) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaUser(o.MqlRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}

		return nil
	}

	err = appendEntry(slice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var slice []*okta.User
		resp, err = resp.Next(ctx, &slice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(slice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil

}

func (o *mqlOktaGroup) roles() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	groupID := o.Id.Data
	slice, resp, err := client.Group.ListGroupAssignedRoles(ctx, groupID, query.NewQueryParams(query.WithLimit(queryLimit)))

	if len(slice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.Role) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaRole(o.MqlRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}

		return nil
	}

	err = appendEntry(slice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var slice []*okta.Role
		resp, err = resp.Next(ctx, &slice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(slice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOkta) groupRules() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.Group.ListGroupRules(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.GroupRule) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaGroupRule(o.MqlRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}

		return nil
	}

	err = appendEntry(slice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var slice []*okta.GroupRule
		resp, err = resp.Next(ctx, &slice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(slice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaGroupRule(runtime *plugin.Runtime, entry *okta.GroupRule) (interface{}, error) {

	return CreateResource(runtime, "okta.groupRule", map[string]*llx.RawData{
		"id":     llx.StringData(entry.Id),
		"name":   llx.StringData(entry.Name),
		"status": llx.StringData(entry.Status),
		"type":   llx.StringData(entry.Type),
	})
}

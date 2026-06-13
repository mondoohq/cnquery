// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

func (o *mqlOkta) groups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.GroupAPI.ListGroups(ctx).Limit(queryLimit).Execute()
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []okta.Group) error {
		for i := range datalist {
			r, err := newMqlOktaGroup(o.MqlRuntime, &datalist[i])
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
		var slice []okta.Group
		resp, err = resp.Next(&slice)
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

func newMqlOktaGroup(runtime *plugin.Runtime, entry *okta.Group) (any, error) {
	profile, err := convert.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	var name, description string
	if entry.Profile != nil {
		name = oktaStr(entry.Profile.Name)
		description = oktaStr(entry.Profile.Description)
	}

	return CreateResource(runtime, "okta.group", map[string]*llx.RawData{
		"id":                    llx.StringData(oktaStr(entry.Id)),
		"name":                  llx.StringData(name),
		"description":           llx.StringData(description),
		"type":                  llx.StringData(oktaStr(entry.Type)),
		"created":               llx.TimeDataPtr(entry.Created),
		"lastMembershipUpdated": llx.TimeDataPtr(entry.LastMembershipUpdated),
		"lastUpdated":           llx.TimeDataPtr(entry.LastUpdated),
		"profile":               llx.DictData(profile),
	})
}

func (o *mqlOktaGroup) id() (string, error) {
	return "okta.group/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaGroup) members() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	groupID := o.Id.Data
	slice, resp, err := client.GroupAPI.ListGroupUsers(ctx, groupID).Limit(queryLimit).Execute()
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []okta.GroupMember) error {
		for i := range datalist {
			r, err := newMqlOktaUser(o.MqlRuntime, &datalist[i])
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
		var slice []okta.GroupMember
		resp, err = resp.Next(&slice)
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

func (o *mqlOktaGroup) roles() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	groupID := o.Id.Data
	slice, resp, err := client.RoleAssignmentAPI.ListGroupAssignedRoles(ctx, groupID).Execute()
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []okta.Role) error {
		for i := range datalist {
			r, err := newMqlOktaRole(o.MqlRuntime, &datalist[i])
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
		var slice []okta.Role
		resp, err = resp.Next(&slice)
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

func (o *mqlOkta) groupRules() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.GroupAPI.ListGroupRules(ctx).Limit(queryLimit).Execute()
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []okta.GroupRule) error {
		for i := range datalist {
			r, err := newMqlOktaGroupRule(o.MqlRuntime, &datalist[i])
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
		var slice []okta.GroupRule
		resp, err = resp.Next(&slice)
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

func newMqlOktaGroupRule(runtime *plugin.Runtime, entry *okta.GroupRule) (any, error) {
	return CreateResource(runtime, "okta.groupRule", map[string]*llx.RawData{
		"id":     llx.StringData(oktaStr(entry.Id)),
		"name":   llx.StringData(oktaStr(entry.Name)),
		"status": llx.StringData(oktaStr(entry.Status)),
		"type":   llx.StringData(oktaStr(entry.Type)),
	})
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/slack/connection"
)

func (s *mqlSlack) userGroups() ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()
	if client == nil {
		return nil, errors.New("cannot retrieve new data while using a mock connection")
	}

	// requires usergroups:read scope
	ctx := context.Background()
	groups, err := client.GetUserGroupsContext(ctx,
		slack.GetUserGroupsOptionIncludeCount(true),
		slack.GetUserGroupsOptionIncludeDisabled(true),
	)
	if err != nil {
		return nil, err
	}
	var list []interface{}
	for i := range groups {
		mqlGroup, err := newMqlSlackUserGroup(s.MqlRuntime, groups[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlGroup)
	}

	return list, nil
}

func newMqlSlackUserGroup(runtime *plugin.Runtime, userGroup slack.UserGroup) (interface{}, error) {
	dateCreate := userGroup.DateCreate.Time()
	dateUpdate := userGroup.DateUpdate.Time()
	dateDelete := userGroup.DateDelete.Time()

	createdBy, err := NewResource(runtime, "slack.user", map[string]*llx.RawData{
		"id": llx.StringData(userGroup.CreatedBy),
	})
	if err != nil {
		return nil, err
	}

	updatedBy, err := NewResource(runtime, "slack.user", map[string]*llx.RawData{
		"id": llx.StringData(userGroup.UpdatedBy),
	})
	if err != nil {
		return nil, err
	}

	deletedBy, err := NewResource(runtime, "slack.user", map[string]*llx.RawData{
		"id": llx.StringData(userGroup.DeletedBy),
	})
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "slack.userGroup", map[string]*llx.RawData{
		"id":          llx.StringData(userGroup.ID),
		"teamId":      llx.StringData(userGroup.TeamID),
		"name":        llx.StringData(userGroup.Name),
		"description": llx.StringData(userGroup.Description),
		"handle":      llx.StringData(userGroup.Handle),
		"isExternal":  llx.BoolData(userGroup.IsExternal),
		"created":     llx.TimeData(dateCreate),
		"updated":     llx.TimeData(dateUpdate),
		"deleted":     llx.TimeData(dateDelete),
		"createdBy":   llx.ResourceData(createdBy, "slack.user"),
		"updatedBy":   llx.ResourceData(updatedBy, "slack.user"),
		"deletedBy":   llx.ResourceData(deletedBy, "slack.user"),
		"userCount":   llx.IntData(int64(userGroup.UserCount)),
	})
}

func (x *mqlSlackUserGroup) id() (string, error) {
	return "slack.userGroup/" + x.Id.Data, nil
}

func (s *mqlSlackUserGroup) members() ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()

	userID := s.Id.Data

	var list []interface{}

	members, err := client.GetUserGroupMembers(userID)
	if err != nil {
		return nil, err
	}

	for i := range members {
		user, err := NewResource(s.MqlRuntime, "slack.user", map[string]*llx.RawData{
			"id": llx.StringData(members[i]),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, user)
	}

	return list, nil
}

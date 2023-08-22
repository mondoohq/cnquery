// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slack

import (
	"context"

	"github.com/slack-go/slack"

	"go.mondoo.com/cnquery/resources"
)

func (s *mqlSlack) GetUserGroups() ([]interface{}, error) {
	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	// requires usergroups:read scope
	groups, err := client.GetUserGroupsContext(ctx,
		slack.GetUserGroupsOptionIncludeCount(true),
		slack.GetUserGroupsOptionIncludeDisabled(true),
	)
	if err != nil {
		return nil, err
	}
	var list []interface{}
	for i := range groups {
		mqlGroup, err := newMqlSlackUserGroup(s.MotorRuntime, groups[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlGroup)
	}

	return list, nil
}

func newMqlSlackUserGroup(runtime *resources.Runtime, userGroup slack.UserGroup) (interface{}, error) {
	dateCreate := userGroup.DateCreate.Time()
	dateUpdate := userGroup.DateUpdate.Time()
	dateDelete := userGroup.DateDelete.Time()

	createdBy, err := runtime.CreateResource("slack.user",
		"id", userGroup.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	updatedBy, err := runtime.CreateResource("slack.user",
		"id", userGroup.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	deletedBy, err := runtime.CreateResource("slack.user",
		"id", userGroup.DeletedBy,
	)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("slack.userGroup",
		"id", userGroup.ID,
		"teamId", userGroup.TeamID,
		"name", userGroup.Name,
		"description", userGroup.Description,
		"handle", userGroup.Handle,
		"isExternal", userGroup.IsExternal,
		"created", &dateCreate,
		"updated", &dateUpdate,
		"deleted", &dateDelete,
		"createdBy", createdBy,
		"updatedBy", updatedBy,
		"deletedBy", deletedBy,
		"userCount", int64(userGroup.UserCount),
	)
}

func (o *mqlSlackUserGroup) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "slack.userGroup/" + id, nil
}

func (s *mqlSlackUserGroup) GetMembers() ([]interface{}, error) {
	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	userID, err := s.Id()
	if err != nil {
		return nil, err
	}

	var list []interface{}

	client := op.Client()
	members, err := client.GetUserGroupMembers(userID)
	if err != nil {
		return nil, err
	}

	for i := range members {
		user, err := s.MotorRuntime.CreateResource("slack.user",
			"id", members[i],
		)
		if err != nil {
			return nil, err
		}
		list = append(list, user)
	}

	return list, nil
}

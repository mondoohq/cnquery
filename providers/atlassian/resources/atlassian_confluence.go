// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/confluence"
)

func (a *mqlAtlassianConfluence) id() (string, error) {
	return "confluence", nil
}

func (a *mqlAtlassianConfluence) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*confluence.ConfluenceConnection)
	confluence := conn.Client()
	cql := "type = user"
	users, _, err := confluence.Search.Users(context.Background(), cql, 0, 1000, nil)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, user := range users.Results {
		mqlAtlassianConfluenceUser, err := CreateResource(a.MqlRuntime, "atlassian.confluence.user",
			map[string]*llx.RawData{
				"id":   llx.StringData(user.User.AccountID),
				"type": llx.StringData(user.User.AccountType),
				"name": llx.StringData(user.User.DisplayName),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianConfluenceUser)
	}
	return res, nil
}

func (a *mqlAtlassianConfluenceUser) id() (string, error) {
	return a.Id.Data, nil
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/atlassian/connection/scim"
)

func (a *mqlAtlassianScim) id() (string, error) {
	return "scim", nil
}

func (a *mqlAtlassianScim) users() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*scim.ScimConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow scim access")
	}
	admin := conn.Client()
	directoryID := conn.Directory()
	scimUsers, _, err := admin.SCIM.User.Gets(context.Background(), directoryID, nil, 0, 1000)
	if err != nil {
		return nil, err
	}
	res := []any{}
	for _, scimUser := range scimUsers.Resources {
		mqlAtlassianAdminSCIMuser, err := CreateResource(a.MqlRuntime, "atlassian.scim.user",
			map[string]*llx.RawData{
				"id":           llx.StringData(scimUser.ID),
				"name":         llx.StringData(scimUser.Name.Formatted),
				"displayName":  llx.StringData(scimUser.DisplayName),
				"organization": llx.StringData(scimUser.Organization),
				"title":        llx.StringData(scimUser.Title),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminSCIMuser)
	}
	return res, nil
}

func (a *mqlAtlassianScim) groups() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*scim.ScimConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow scim access")
	}
	admin := conn.Client()
	directoryID := conn.Directory()
	scimGroup, _, err := admin.SCIM.Group.Gets(context.Background(), directoryID, "", 0, 1000)
	if err != nil {
		return nil, err
	}
	res := []any{}
	for _, scimGroup := range scimGroup.Resources {
		mqlAtlassianAdminSCIMgroup, err := CreateResource(a.MqlRuntime, "atlassian.scim.group",
			map[string]*llx.RawData{
				"id":   llx.StringData(scimGroup.ID),
				"name": llx.StringData(scimGroup.DisplayName),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminSCIMgroup)
	}
	return res, nil
}

func (a *mqlAtlassianScimUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianScimGroup) id() (string, error) {
	return a.Id.Data, nil
}

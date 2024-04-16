// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/google-workspace/connection"
	"go.mondoo.com/cnquery/v11/types"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) roles() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	directoryService, err := directoryService(conn, directory.AdminDirectoryRolemanagementReadonlyScope)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	groups, err := directoryService.Roles.List(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}
	for {
		for i := range groups.Items {
			r, err := newMqlGoogleWorkspaceRole(g.MqlRuntime, groups.Items[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if groups.NextPageToken == "" {
			break
		}

		groups, err = directoryService.Roles.List(conn.CustomerID()).PageToken(groups.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceRole(runtime *plugin.Runtime, entry *directory.Role) (interface{}, error) {
	privileges, err := convert.JsonToDictSlice(entry.RolePrivileges)
	if err != nil {
		return nil, err
	}
	return CreateResource(runtime, "googleworkspace.role", map[string]*llx.RawData{
		"id":               llx.IntData(entry.RoleId),
		"name":             llx.StringData(entry.RoleName),
		"description":      llx.StringData(entry.RoleDescription),
		"isSystemRole":     llx.BoolData(entry.IsSystemRole),
		"isSuperAdminRole": llx.BoolData(entry.IsSuperAdminRole),
		"privileges":       llx.ArrayData(privileges, types.Any),
	})
}

func (g *mqlGoogleworkspaceRole) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "googleworkspace.role/" + strconv.FormatInt(id, 10), nil
}

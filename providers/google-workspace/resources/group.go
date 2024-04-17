// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"go.mondoo.com/cnquery/v11/providers/google-workspace/connection"
	"go.mondoo.com/cnquery/v11/types"
	"google.golang.org/api/groupssettings/v1"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

func (g *mqlGoogleworkspace) groups() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	directoryService, err := directoryService(conn, directory.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	groups, err := directoryService.Groups.List().Customer(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range groups.Groups {
			r, err := newMqlGoogleWorkspaceGroup(g.MqlRuntime, groups.Groups[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if groups.NextPageToken == "" {
			break
		}

		groups, err = directoryService.Groups.List().Customer(conn.CustomerID()).PageToken(groups.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceGroup(runtime *plugin.Runtime, entry *directory.Group) (interface{}, error) {
	return CreateResource(runtime, "googleworkspace.group", map[string]*llx.RawData{
		"id":                 llx.StringData(entry.Id),
		"name":               llx.StringData(entry.Name),
		"email":              llx.StringData(entry.Email),
		"description":        llx.StringData(entry.Description),
		"aliases":            llx.ArrayData(convert.SliceAnyToInterface[string](entry.Aliases), types.Any),
		"directMembersCount": llx.IntData(entry.DirectMembersCount),
		"adminCreated":       llx.BoolData(entry.AdminCreated),
	})
}

func (g *mqlGoogleworkspaceGroup) id() (string, error) {
	return "googleworkspace.group/" + g.Id.Data, g.Id.Error
}

func (g *mqlGoogleworkspaceGroup) members() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	client, err := conn.Client()
	if err != nil {
		return nil, err
	}

	directoryService, err := directory.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	id := g.Id.Data

	res := []interface{}{}

	members, err := directoryService.Members.List(id).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range members.Members {
			r, err := newMqlGoogleWorkspaceMember(g.MqlRuntime, members.Members[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if members.NextPageToken == "" {
			break
		}

		members, err = directoryService.Members.List(id).PageToken(members.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceMember(runtime *plugin.Runtime, entry *directory.Member) (interface{}, error) {
	return CreateResource(runtime, "googleworkspace.member", map[string]*llx.RawData{
		"id":     llx.StringData(entry.Id),
		"email":  llx.StringData(entry.Email),
		"status": llx.StringData(entry.Status),
		"type":   llx.StringData(entry.Type),
	})
}

func (g *mqlGoogleworkspaceMember) id() (string, error) {
	return "googleworkspace.member/" + g.Id.Data, g.Id.Error
}

func (g *mqlGoogleworkspaceMember) user() (*mqlGoogleworkspaceUser, error) {
	if g.Email.Error != nil {
		return nil, g.Email.Error
	}
	email := g.Email.Data
	if g.Type.Error != nil {
		return nil, g.Type.Error
	}
	typ := g.Type.Data

	if strings.ToLower(typ) != "user" {
		return nil, nil
	}

	obj, err := CreateResource(g.MqlRuntime, "googleworkspace", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	gws := obj.(*mqlGoogleworkspace)

	if gws.Users.Error != nil {
		return nil, gws.Users.Error
	}
	users := gws.Users.Data

	for i := range users {
		user := users[i].(*mqlGoogleworkspaceUser)
		if user.PrimaryEmail.Error != nil {
			return nil, user.PrimaryEmail.Error
		}
		primaryEmail := user.PrimaryEmail.Data
		if primaryEmail == email {
			return user, nil
		}
	}
	return nil, nil
}

func (g *mqlGoogleworkspaceGroup) settings() (interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	service, err := groupSettingsService(conn, groupssettings.AppsGroupsSettingsScope)
	if err != nil {
		return nil, err
	}

	if g.Email.Error != nil {
		return nil, g.Email.Error
	}
	email := g.Email.Data

	settings, err := service.Groups.Get(email).Do()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(settings)
}

func (g *mqlGoogleworkspaceGroup) securitySettings() (interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	service, err := cloudIdentityService(conn, cloudidentity.CloudIdentityGroupsReadonlyScope)
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	groupId := g.Id.Data

	securitySettings, err := service.Groups.GetSecuritySettings(`groups/` + groupId + `/securitySettings`).ReadMask("*").Do()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(securitySettings)
}

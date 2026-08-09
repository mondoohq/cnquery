// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// mapProject builds the resource args for an openai.project. Both the
// collection path and the single-object init share it so the two paths cannot
// diverge.
func mapProject(p openai.Project) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"__id":       llx.StringData(p.ID),
		"id":         llx.StringData(p.ID),
		"name":       llx.StringData(p.Name),
		"status":     llx.StringData(string(p.Status)),
		"createdAt":  llx.TimeDataPtr(unixToNullableTime(p.CreatedAt)),
		"archivedAt": llx.TimeDataPtr(unixToNullableTime(p.ArchivedAt)),
	}
}

func (r *mqlOpenai) projects() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.projects")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.ListAutoPaging(ctx, openai.AdminOrganizationProjectListParams{})
	var res []any
	for iter.Next() {
		p := iter.Current()
		mqlProject, err := CreateResource(r.MqlRuntime, "openai.project", mapProject(p))
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProject)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return res, nil
}

func initOpenaiProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idRaw, ok := args["id"]
	if !ok || idRaw == nil || idRaw.Value == nil {
		return args, nil, nil
	}
	projectID, ok := idRaw.Value.(string)
	if !ok || projectID == "" {
		return args, nil, nil
	}

	conn := openaiConn(runtime)
	client, err := adminPlaneClient(conn, "openai.project")
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		return nil, nil, fmt.Errorf("cannot fetch project %s: no admin API key configured", projectID)
	}
	p, err := client.Admin.Organization.Projects.Get(context.Background(), projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get project %s: %w", projectID, err)
	}
	return mapProject(*p), nil, nil
}

func (r *mqlOpenaiProject) apiKeys() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.project.apiKeys")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.APIKeys.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectAPIKeyListParams{})
	var res []any
	for iter.Next() {
		k := iter.Current()

		ownerType := k.Owner.Type
		var ownerName, ownerId string
		switch ownerType {
		case "user":
			ownerName = k.Owner.User.Email
			ownerId = k.Owner.User.ID
		case "service_account":
			ownerName = k.Owner.ServiceAccount.Name
			ownerId = k.Owner.ServiceAccount.ID
		}

		mqlKey, err := CreateResource(r.MqlRuntime, "openai.project.apiKey", map[string]*llx.RawData{
			"__id":          llx.StringData(k.ID),
			"id":            llx.StringData(k.ID),
			"name":          llx.StringData(k.Name),
			"redactedValue": llx.StringData(k.RedactedValue),
			"createdAt":     llx.TimeDataPtr(unixToNullableTime(k.CreatedAt)),
			"lastUsedAt":    llx.TimeDataPtr(unixToNullableTime(k.LastUsedAt)),
			"ownerType":     llx.StringData(ownerType),
			"ownerName":     llx.StringData(ownerName),
			"ownerId":       llx.StringData(ownerId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKey)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project API keys: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) serviceAccounts() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.project.serviceAccounts")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.ServiceAccounts.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectServiceAccountListParams{})
	var res []any
	for iter.Next() {
		sa := iter.Current()

		mqlSA, err := CreateResource(r.MqlRuntime, "openai.project.serviceAccount", map[string]*llx.RawData{
			"__id":      llx.StringData(sa.ID),
			"id":        llx.StringData(sa.ID),
			"name":      llx.StringData(sa.Name),
			"role":      llx.StringData(string(sa.Role)),
			"createdAt": llx.TimeDataPtr(unixToNullableTime(sa.CreatedAt)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSA)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project service accounts: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) users() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.project.users")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.Users.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectUserListParams{})
	var res []any
	for iter.Next() {
		u := iter.Current()

		// the same user belongs to several projects, so the membership is keyed
		// by project as well as by user
		mqlUser, err := CreateResource(r.MqlRuntime, "openai.project.user", map[string]*llx.RawData{
			"__id":    llx.StringData(r.Id.Data + "/" + u.ID),
			"id":      llx.StringData(u.ID),
			"email":   llx.StringData(u.Email),
			"name":    llx.StringData(u.Name),
			"role":    llx.StringData(u.Role),
			"addedAt": llx.TimeDataPtr(unixToNullableTime(u.AddedAt)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project members: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProjectUser) user() (*mqlOpenaiOrganizationUser, error) {
	return resolveOrganizationUser(r.MqlRuntime, r.Id.Data)
}

func (r *mqlOpenaiProject) groups() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.project.groups")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.Groups.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectGroupListParams{})
	var res []any
	for iter.Next() {
		pg := iter.Current()
		mqlGroup, err := NewResource(r.MqlRuntime, "openai.group", map[string]*llx.RawData{
			"id": llx.StringData(pg.GroupID),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project groups: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) roles() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.project.roles")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.Roles.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectRoleListParams{})
	var res []any
	for iter.Next() {
		role := iter.Current()
		mqlRole, err := CreateResource(r.MqlRuntime, "openai.role",
			roleArgs(role.ID, role.Name, role.Description, role.ResourceType, role.Permissions, role.PredefinedRole))
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRole)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project roles: %w", err)
	}
	return res, nil
}

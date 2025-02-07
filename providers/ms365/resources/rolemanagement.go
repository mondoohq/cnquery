// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

var roledefinitionsSelectFields = []string{
	"id",
	"description",
	"displayName",
	"isBuiltIn",
	"isEnabled",
	"rolePermissions",
	"templateId",
	"version",
}

func (a *mqlMicrosoftRoles) list() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	opts := &rolemanagement.DirectoryRoleDefinitionsRequestBuilderGetRequestConfiguration{
		QueryParameters: &rolemanagement.DirectoryRoleDefinitionsRequestBuilderGetQueryParameters{
			Select: roledefinitionsSelectFields,
		},
	}

	if a.Search.State == plugin.StateIsSet || a.Filter.State == plugin.StateIsSet {
		// search and filter requires this header
		headers := abstractions.NewRequestHeaders()
		headers.Add("ConsistencyLevel", "eventual")
		opts.Headers = headers

		if a.Search.State == plugin.StateIsSet {
			log.Debug().
				Str("search", a.Search.Data).
				Msg("microsoft.roles.list.search set")
			search, err := parseSearch(a.Search.Data)
			if err != nil {
				return nil, err
			}
			opts.QueryParameters.Search = &search
		}
		if a.Filter.State == plugin.StateIsSet {
			log.Debug().
				Str("filter", a.Filter.Data).
				Msg("microsoft.roles.list.filter set")
			opts.QueryParameters.Filter = &a.Filter.Data
			count := true
			opts.QueryParameters.Count = &count
		}
	}

	resp, err := graphClient.
		RoleManagement().
		Directory().
		RoleDefinitions().
		Get(ctx, opts)
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	roles := resp.GetValue()
	for _, role := range roles {
		rolePermissions, err := convert.JsonToDictSlice(newUnifiedRolePermissions(role.GetRolePermissions()))
		if err != nil {
			return nil, err
		}
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.rolemanagement.roledefinition",
			map[string]*llx.RawData{
				"id":              llx.StringDataPtr(role.GetId()),
				"description":     llx.StringDataPtr(role.GetDescription()),
				"displayName":     llx.StringDataPtr(role.GetDisplayName()),
				"isBuiltIn":       llx.BoolDataPtr(role.GetIsBuiltIn()),
				"isEnabled":       llx.BoolDataPtr(role.GetIsEnabled()),
				"rolePermissions": llx.ArrayData(rolePermissions, types.Any),
				"templateId":      llx.StringDataPtr(role.GetTemplateId()),
				"version":         llx.StringDataPtr(role.GetVersion()),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (a *mqlMicrosoft) roles() (*mqlMicrosoftRoles, error) {
	resource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft.roles", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftRoles), nil
}

func initMicrosoftRoles(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	args["__id"] = newListResourceIdFromArguments("microsoft.roles", args)
	resource, err := runtime.CreateResource(runtime, "microsoft.roles", args)
	if err != nil {
		return args, nil, err
	}

	return args, resource.(*mqlMicrosoftRoles), nil
}

func (m *mqlMicrosoftRolemanagementRoledefinition) id() (string, error) {
	return m.Id.Data, nil
}

// Deprecated: use mqlMicrosoft roles() instead
func (m *mqlMicrosoftRolemanagementRoleassignment) id() (string, error) {
	return m.Id.Data, nil
}

// Deprecated: use mqlMicrosoft roles() instead
func (a *mqlMicrosoftRolemanagement) roleDefinitions() (*mqlMicrosoftRoles, error) {
	resource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft.roles", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftRoles), nil
}

func (a *mqlMicrosoftRolemanagementRoledefinition) assignments() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	roleDefinitionId := a.Id.Data
	filter := "roleDefinitionId eq '" + roleDefinitionId + "'"
	requestConfig := &rolemanagement.DirectoryRoleAssignmentsRequestBuilderGetRequestConfiguration{
		QueryParameters: &rolemanagement.DirectoryRoleAssignmentsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Expand: []string{"principal"},
		},
	}
	ctx := context.Background()
	resp, err := graphClient.RoleManagement().Directory().RoleAssignments().Get(ctx, requestConfig)
	if err != nil {
		return nil, transformError(err)
	}

	roleAssignments := resp.GetValue()
	res := []interface{}{}
	for _, roleAssignment := range roleAssignments {
		principal, err := convert.JsonToDict(newDirectoryPrincipal(roleAssignment.GetPrincipal()))
		if err != nil {
			return nil, err
		}
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.rolemanagement.roleassignment",
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(roleAssignment.GetId()),
				"roleDefinitionId": llx.StringDataPtr(roleAssignment.GetRoleDefinitionId()),
				"principalId":      llx.StringDataPtr(roleAssignment.GetPrincipalId()),
				"principal":        llx.DictData(principal),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}
	return res, nil
}

// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (m *mqlMicrosoftRolemanagementRoledefinition) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftRolemanagementRoleassignment) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoftRolemanagement) roleDefinitions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.RoleManagement().Directory().RoleDefinitions().Get(ctx, &rolemanagement.DirectoryRoleDefinitionsRequestBuilderGetRequestConfiguration{})
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

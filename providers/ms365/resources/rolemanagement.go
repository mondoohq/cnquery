// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"log"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"

	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func fetchRoles(runtime *plugin.Runtime) ([]interface{}, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	resp, err := graphClient.RoleManagement().Directory().RoleDefinitions().Get(ctx, &rolemanagement.DirectoryRoleDefinitionsRequestBuilderGetRequestConfiguration{
		QueryParameters: &rolemanagement.DirectoryRoleDefinitionsRequestBuilderGetQueryParameters{
			Select: []string{"id", "description", "displayName", "isBuiltIn", "isEnabled", "rolePermissions", "templateId", "version"},
		},
	})
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
		mqlResource, err := CreateResource(runtime, "microsoft.rolemanagement.roledefinition",
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

func (a *mqlMicrosoft) roles() ([]interface{}, error) {
	return fetchRoles(a.MqlRuntime)
}

func (m *mqlMicrosoftRolemanagementRoledefinition) id() (string, error) {
	return m.Id.Data, nil
}

// Deprecated: use mqlMicrosoft roles() instead
func (m *mqlMicrosoftRolemanagementRoleassignment) id() (string, error) {
	return m.Id.Data, nil
}

// Deprecated: use mqlMicrosoft roles() instead
func (a *mqlMicrosoftRolemanagement) roleDefinitions() ([]interface{}, error) {
	return fetchRoles(a.MqlRuntime)
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

// Related to Delegated Admin Portal under Roles & admin in Entra ID
func (a *mqlMicrosoftAdminPortal) delegatedAdminPartners() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	partnersResp, err := graphClient.TenantRelationships().DelegatedAdminRelationships().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var partnerDetails []interface{}
	for _, partner := range partnersResp.GetValue() {
		partnerId := partner.GetId()
		displayName := partner.GetDisplayName()

		if partnerId != nil && displayName != nil {
			partnerInfo, err := CreateResource(a.MqlRuntime, "microsoft.adminPortal.delegatedAdminPartner",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(partnerId),
					"displayName": llx.StringDataPtr(displayName),
				})
			if err != nil {
				return nil, err
			}
			partnerDetails = append(partnerDetails, partnerInfo)
		}
	}

	if len(partnerDetails) == 0 {
		log.Println("No delegated admin partners are defined.")
		return nil, nil
	}

	return partnerDetails, nil
}

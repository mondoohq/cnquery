// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/rolemanagement"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/ms365/connection"
	"go.mondoo.com/mql/v13/types"
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

func (a *mqlMicrosoftRoles) list() ([]any, error) {
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
	roles, err := iterate[models.UnifiedRoleDefinitionable](ctx, resp, graphClient.GetAdapter(), models.CreateUnifiedRoleDefinitionCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, role := range roles {
		mqlResource, err := newMqlMicrosoftRoleDefinition(a.MqlRuntime, role)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

// newMqlMicrosoftRoleDefinition builds a microsoft.rolemanagement.roledefinition
// resource from a Graph unified role definition.
func newMqlMicrosoftRoleDefinition(runtime *plugin.Runtime, role models.UnifiedRoleDefinitionable) (*mqlMicrosoftRolemanagementRoledefinition, error) {
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
	return mqlResource.(*mqlMicrosoftRolemanagementRoledefinition), nil
}

// initMicrosoftRolemanagementRoledefinition resolves a single role definition
// by its ID, enabling typed role references from Conditional Access conditions
// and role assignments.
func initMicrosoftRolemanagementRoledefinition(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// only resolve when handed a bare id; a fully populated role definition
	// passes through untouched
	if len(args) != 1 {
		return args, nil, nil
	}
	rawId, ok := args["id"]
	if !ok {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	role, err := graphClient.
		RoleManagement().
		Directory().
		RoleDefinitions().
		ByUnifiedRoleDefinitionId(rawId.Value.(string)).
		Get(ctx, nil)
	if err != nil {
		return nil, nil, transformError(err)
	}

	mqlRole, err := newMqlMicrosoftRoleDefinition(runtime, role)
	if err != nil {
		return nil, nil, err
	}
	return nil, mqlRole, nil
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

// roleDefinition resolves the role definition this assignment grants.
func (m *mqlMicrosoftRolemanagementRoleassignment) roleDefinition() (*mqlMicrosoftRolemanagementRoledefinition, error) {
	id := m.RoleDefinitionId.Data
	if id == "" {
		m.RoleDefinition.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(m.MqlRuntime, "microsoft.rolemanagement.roledefinition", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftRolemanagementRoledefinition), nil
}

// Deprecated: use mqlMicrosoft roles() instead
func (a *mqlMicrosoftRolemanagement) roleDefinitions() (*mqlMicrosoftRoles, error) {
	resource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "microsoft.roles", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftRoles), nil
}

func (a *mqlMicrosoftRolemanagementRoledefinition) assignments() ([]any, error) {
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
	roleAssignments, err := iterate[models.UnifiedRoleAssignmentable](ctx, resp, graphClient.GetAdapter(), models.CreateUnifiedRoleAssignmentCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	res := []any{}
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

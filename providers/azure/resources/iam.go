// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	authorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAzureSubscription) iam() (*mqlAzureSubscriptionAuthorizationService, error) {
	svc, err := NewResource(a.MqlRuntime, "azure.subscription.authorizationService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	authSvc := svc.(*mqlAzureSubscriptionAuthorizationService)
	return authSvc, nil
}

// Deprecated: use iam instead
func (a *mqlAzureSubscription) authorization() (*mqlAzureSubscriptionAuthorizationService, error) {
	return a.iam()
}

func (a *mqlAzureSubscriptionAuthorizationService) id() (string, error) {
	return "azure.subscription.authorization/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionAuthorizationService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

// Deprecated: use roles instead
func (a *mqlAzureSubscriptionAuthorizationService) roleDefinitions() ([]interface{}, error) {
	return a.roles()
}

func (a *mqlAzureSubscriptionAuthorizationService) roles() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := authorization.NewRoleDefinitionsClient(token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// we're interested in subscription-level role definitions, so we scope this to the subscription,
	// on which this connection is running
	scope := fmt.Sprintf("/subscriptions/%s", subId)
	pager := client.NewListPager(scope, &authorization.RoleDefinitionsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, roleDef := range page.Value {
			roleType := convert.ToValue(roleDef.Properties.RoleType)
			isCustom := roleType == "CustomRole"
			scopes := []interface{}{}
			for _, s := range roleDef.Properties.AssignableScopes {
				if s != nil {
					scopes = append(scopes, *s)
				}
			}
			permissions := []interface{}{}
			for idx, p := range roleDef.Properties.Permissions {
				id := fmt.Sprintf("%s/azure.subscription.authorizationService.roleDefinition.permission/%d", *roleDef.ID, idx)
				permission, err := newMqlRolePermission(a.MqlRuntime, id, p)
				if err != nil {
					return nil, err
				}
				permissions = append(permissions, permission)
			}
			mqlRoleDefinition, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.roleDefinition",
				map[string]*llx.RawData{
					"__id":        llx.StringDataPtr(roleDef.ID),
					"id":          llx.StringDataPtr(roleDef.ID),
					"name":        llx.StringDataPtr(roleDef.Properties.RoleName),
					"description": llx.StringDataPtr(roleDef.Properties.Description),
					"type":        llx.StringData(roleType),
					"isCustom":    llx.BoolData(isCustom),
					"scopes":      llx.ArrayData(scopes, types.String),
					"permissions": llx.ArrayData(permissions, types.ResourceLike),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRoleDefinition)
		}
	}
	return res, nil
}

func newMqlRolePermission(runtime *plugin.Runtime, id string, permission *authorization.Permission) (interface{}, error) {
	allowedActions := []interface{}{}
	deniedActions := []interface{}{}
	allowedDataActions := []interface{}{}
	deniedDataActions := []interface{}{}

	for _, a := range permission.Actions {
		if a != nil {
			allowedActions = append(allowedActions, *a)
		}
	}
	for _, a := range permission.NotActions {
		if a != nil {
			deniedActions = append(deniedActions, *a)
		}
	}
	for _, a := range permission.DataActions {
		if a != nil {
			allowedDataActions = append(allowedDataActions, *a)
		}
	}
	for _, a := range permission.NotDataActions {
		if a != nil {
			deniedDataActions = append(deniedDataActions, *a)
		}
	}

	p, err := CreateResource(runtime, "azure.subscription.authorizationService.roleDefinition.permission",
		map[string]*llx.RawData{
			"__id":               llx.StringData(id),
			"id":                 llx.StringData(id),
			"allowedActions":     llx.ArrayData(allowedActions, types.String),
			"deniedActions":      llx.ArrayData(deniedActions, types.String),
			"allowedDataActions": llx.ArrayData(allowedDataActions, types.String),
			"deniedDataActions":  llx.ArrayData(deniedDataActions, types.String),
		})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (a *mqlAzureSubscriptionAuthorizationService) roleAssignments() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := authorization.NewRoleAssignmentsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	// we're interested in subscription-level role definitions, so we scope this to the subscription,
	// on which this connection is running
	pager := client.NewListForSubscriptionPager(&authorization.RoleAssignmentsClientListForSubscriptionOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, roleAssignment := range page.Value {
			mqlRoleAssignment, err := newMqlRoleAssignment(a.MqlRuntime, roleAssignment)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRoleAssignment)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionAuthorizationServiceRoleAssignmentInternal struct {
	roleDefinitionId string
}

func newMqlRoleAssignment(runtime *plugin.Runtime, roleAssignment *authorization.RoleAssignment) (*mqlAzureSubscriptionAuthorizationServiceRoleAssignment, error) {
	r, err := CreateResource(runtime, "azure.subscription.authorizationService.roleAssignment",
		map[string]*llx.RawData{
			"__id":        llx.StringDataPtr(roleAssignment.ID),
			"id":          llx.StringDataPtr(roleAssignment.Name), // name is the id :-)
			"description": llx.StringDataPtr(roleAssignment.Properties.Description),
			"scope":       llx.StringDataPtr(roleAssignment.Properties.Scope),
			"type":        llx.StringData(string(*roleAssignment.Properties.PrincipalType)),
			"principalId": llx.StringData(*roleAssignment.Properties.PrincipalID),
			"condition":   llx.StringDataPtr(roleAssignment.Properties.Condition),
			"createdAt":   llx.TimeDataPtr(roleAssignment.Properties.CreatedOn),
			"updatedAt":   llx.TimeDataPtr(roleAssignment.Properties.UpdatedOn),
		})
	if err != nil {
		return nil, err
	}

	mqlRoleDefinition := r.(*mqlAzureSubscriptionAuthorizationServiceRoleAssignment)
	if roleAssignment.Properties.RoleDefinitionID != nil {
		mqlRoleDefinition.roleDefinitionId = *roleAssignment.Properties.RoleDefinitionID
	}
	return mqlRoleDefinition, nil
}

func extractSubscriptionID(roleDefinitionID string) (string, error) {
	parts := strings.Split(roleDefinitionID, "/")

	for i, part := range parts {
		if part == "subscriptions" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}

	return "", fmt.Errorf("subscription ID not found in role definition ID")
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleAssignment) role() (*mqlAzureSubscriptionAuthorizationServiceRoleDefinition, error) {
	if a.roleDefinitionId == "" {
		return nil, nil
	}

	// extract subscription id from role definition id
	subId, err := extractSubscriptionID(a.roleDefinitionId)
	if err != nil {
		return nil, err
	}

	r, err := CreateResource(a.MqlRuntime, "azure.subscription", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(subId),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlAzureSubscription)
	iamResource := mqlResource.GetIam().Data
	roles := iamResource.GetRoles().Data
	for i := range roles {
		role := roles[i].(*mqlAzureSubscriptionAuthorizationServiceRoleDefinition)
		if role.__id == a.roleDefinitionId {
			return role, nil
		}
	}

	return nil, errors.New("role definition not found")
}

func (a *mqlAzureSubscriptionAuthorizationService) managedIdentities() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armmsi.NewUserAssignedIdentitiesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// list all role assignemnts since we need to attach them to the managed identities
	roleAssignments := a.GetRoleAssignments().Data

	// list user assigned identities
	pager := client.NewListBySubscriptionPager(nil)
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range page.Value {
			mqlManagedIdentity, err := newMqlManagedIdentity(a.MqlRuntime, v)
			if err != nil {
				return nil, err
			}

			// set assigned roles to nil
			mqlManagedIdentity.RoleAssignments = plugin.TValue[[]interface{}]{Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

			assignedRoles := []interface{}{}
			for i := range roleAssignments {
				roleAssignment := roleAssignments[i].(*mqlAzureSubscriptionAuthorizationServiceRoleAssignment)
				if roleAssignment.PrincipalId == mqlManagedIdentity.PrincipalId {
					assignedRoles = append(assignedRoles, roleAssignment)
				}
			}

			if len(assignedRoles) > 0 {
				mqlManagedIdentity.RoleAssignments = plugin.TValue[[]interface{}]{Error: nil, Data: assignedRoles, State: plugin.StateIsSet}
			}

			res = append(res, mqlManagedIdentity)
		}
	}
	return res, nil
}

func newMqlManagedIdentity(runtime *plugin.Runtime, managedIdentity *armmsi.Identity) (*mqlAzureSubscriptionManagedIdentity, error) {
	r, err := CreateResource(runtime, "azure.subscription.managedIdentity",
		map[string]*llx.RawData{
			"__id":        llx.StringDataPtr(managedIdentity.ID),
			"name":        llx.StringDataPtr(managedIdentity.Name),
			"clientId":    llx.StringDataPtr(managedIdentity.Properties.ClientID),
			"principalId": llx.StringDataPtr(managedIdentity.Properties.PrincipalID),
			"tenantId":    llx.StringData(string(*managedIdentity.Properties.TenantID)),
		})
	if err != nil {
		return nil, err
	}

	mqlManagedIdentity := r.(*mqlAzureSubscriptionManagedIdentity)
	return mqlManagedIdentity, nil
}

func (a *mqlAzureSubscriptionManagedIdentity) roleAssignments() ([]interface{}, error) {
	// NOTE: this should never be called since we assign roles during the managed identities query
	return nil, errors.New("could not fetch role assignments for managed identities")
}

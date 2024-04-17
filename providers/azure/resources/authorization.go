// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

	authorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
)

func (a *mqlAzureSubscriptionAuthorizationService) id() (string, error) {
	return "azure.subscription.authorization/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionAuthorizationService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleDefinition) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleDefinitionPermission) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAuthorizationService) roleDefinitions() ([]interface{}, error) {
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
			roleType := convert.ToString(roleDef.Properties.RoleType)
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
				permission, err := azureToMqlPermission(a.MqlRuntime, id, p)
				if err != nil {
					return nil, err
				}
				permissions = append(permissions, permission)
			}
			if isCustom {
				isCustom = true
			}
			mqlRoleDefinition, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.roleDefinition",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(roleDef.ID),
					"name":        llx.StringDataPtr(roleDef.Properties.RoleName),
					"description": llx.StringDataPtr(roleDef.Properties.Description),
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

func azureToMqlPermission(runtime *plugin.Runtime, id string, permission *authorization.Permission) (interface{}, error) {
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

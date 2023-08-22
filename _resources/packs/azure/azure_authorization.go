// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	authorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscriptionAuthorizationService) init(args *resources.Args) (*resources.Args, AzureSubscriptionAuthorizationService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	(*args)["subscriptionId"] = at.SubscriptionID()

	return args, nil, nil
}

func (a *mqlAzureSubscriptionAuthorizationService) id() (string, error) {
	subId, err := a.SubscriptionId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/subscriptions/%s/authorizationService", subId), nil
}

// note: requires Microsoft.Authorization/roleDefinitions/read to read role definitions
func (a *mqlAzureSubscriptionAuthorizationService) GetRoleDefinitions() (interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	client, err := authorization.NewRoleDefinitionsClient(token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	// we're interested in subscription-level role definitions, so we scope this to the subscription,
	// on which this provider is running
	scope := fmt.Sprintf("/subscriptions/%s", at.SubscriptionID())
	pager := client.NewListPager(scope, &authorization.RoleDefinitionsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, roleDef := range page.Value {
			roleType := core.ToString(roleDef.Properties.RoleType)
			isCustom := roleType == "CustomRole"
			scopes := []interface{}{}
			for _, s := range roleDef.Properties.AssignableScopes {
				if s != nil {
					scopes = append(scopes, *s)
				}
			}
			permissions := []interface{}{}
			for idx, p := range roleDef.Properties.Permissions {
				id := fmt.Sprintf("%s/azure.subscription.authorization.roleDefinition.permission/%d", *roleDef.ID, idx)
				permission, err := azureToMqlPermission(a.MotorRuntime, id, p)
				if err != nil {
					return nil, err
				}
				permissions = append(permissions, permission)
			}
			if isCustom {
				isCustom = true
			}
			mqlRoleDefinition, err := a.MotorRuntime.CreateResource("azure.subscription.authorizationService.roleDefinition",
				"id", core.ToString(roleDef.ID),
				"name", core.ToString(roleDef.Properties.RoleName),
				"description", core.ToString(roleDef.Properties.Description),
				"isCustom", isCustom,
				"scopes", scopes,
				"permissions", permissions,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRoleDefinition)
		}
	}
	return res, nil
}

func azureToMqlPermission(runtime *resources.Runtime, id string, permission *authorization.Permission) (interface{}, error) {
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

	p, err := runtime.CreateResource("azure.subscription.authorizationService.roleDefinition.permission",
		"id", id,
		"allowedActions", allowedActions,
		"deniedActions", deniedActions,
		"allowedDataActions", allowedDataActions,
		"deniedDataActions", deniedDataActions)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleDefinition) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleDefinitionPermission) id() (string, error) {
	return a.Id()
}

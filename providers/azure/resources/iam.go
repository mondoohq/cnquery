// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	authorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

// sortedUserAssignedIdentityIDs returns the keys of a UserAssignedIdentities
// map (the ARM resource IDs of the assigned managed identities) in stable
// sorted order. Returns nil when the map is empty.
func sortedUserAssignedIdentityIDs[V any](m map[string]V) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// resolveUserAssignedIdentities resolves a set of user-assigned managed
// identity ARM resource IDs into typed managedIdentity resources. The IDs are
// the keys of an SDK UserAssignedIdentities map. The runtime cache short-
// circuits identities already listed by managedIdentities(); others are
// fetched on demand by their init. Returns an empty slice when none are set.
func resolveUserAssignedIdentities(runtime *plugin.Runtime, ids []string) ([]any, error) {
	res := make([]any, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		mqlIdentity, err := NewResource(runtime, "azure.subscription.managedIdentity",
			map[string]*llx.RawData{"__id": llx.StringData(id)})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIdentity)
	}
	return res, nil
}

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

func (a *mqlAzureSubscriptionAuthorizationService) roles() ([]any, error) {
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
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, roleDef := range page.Value {
			roleType := convert.ToValue(roleDef.Properties.RoleType)
			scopes := []any{}
			for _, s := range roleDef.Properties.AssignableScopes {
				if s != nil {
					scopes = append(scopes, *s)
				}
			}
			mqlRoleDefinition, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.roleDefinition",
				map[string]*llx.RawData{
					"__id":        llx.StringDataPtr(roleDef.ID),
					"id":          llx.StringDataPtr(roleDef.ID),
					"name":        llx.StringDataPtr(roleDef.Properties.RoleName),
					"description": llx.StringDataPtr(roleDef.Properties.Description),
					"type":        llx.StringData(roleType),
					"scopes":      llx.ArrayData(scopes, types.String),
				})
			if err != nil {
				return nil, err
			}
			mqlRole := mqlRoleDefinition.(*mqlAzureSubscriptionAuthorizationServiceRoleDefinition)
			mqlRole.cachePermissions = roleDef.Properties.Permissions
			res = append(res, mqlRole)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionAuthorizationServiceRoleDefinitionInternal struct {
	cachePermissions []*authorization.Permission
}

func (a *mqlAzureSubscriptionAuthorizationServiceRoleDefinition) permissions() ([]any, error) {
	res := []any{}
	for idx, p := range a.cachePermissions {
		if p == nil {
			continue
		}
		id := fmt.Sprintf("%s/azure.subscription.authorizationService.roleDefinition.permission/%d", a.Id.Data, idx)
		permission, err := newMqlRolePermission(a.MqlRuntime, id, p)
		if err != nil {
			return nil, err
		}
		res = append(res, permission)
	}
	return res, nil
}

func newMqlRolePermission(runtime *plugin.Runtime, id string, permission *authorization.Permission) (any, error) {
	allowedActions := []any{}
	deniedActions := []any{}
	allowedDataActions := []any{}
	deniedDataActions := []any{}

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

func (a *mqlAzureSubscriptionAuthorizationService) roleAssignments() ([]any, error) {
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
	res := []any{}
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
	// Properties is a nullable pointer; the args map dereferences it throughout,
	// so normalize to an empty struct to avoid a panic (mirrors the
	// denyAssignments/classicAdministrators paths below).
	if roleAssignment.Properties == nil {
		roleAssignment.Properties = &authorization.RoleAssignmentProperties{}
	}
	principalType := string(convert.ToValue(roleAssignment.Properties.PrincipalType))
	r, err := CreateResource(runtime, "azure.subscription.authorizationService.roleAssignment",
		map[string]*llx.RawData{
			"__id":          llx.StringDataPtr(roleAssignment.ID),
			"id":            llx.StringDataPtr(roleAssignment.Name), // name is the id :-)
			"description":   llx.StringDataPtr(roleAssignment.Properties.Description),
			"scope":         llx.StringDataPtr(roleAssignment.Properties.Scope),
			"type":          llx.StringData(principalType),
			"principalId":   llx.StringDataPtr(roleAssignment.Properties.PrincipalID),
			"principalType": llx.StringData(principalType),
			"condition":     llx.StringDataPtr(roleAssignment.Properties.Condition),
			"createdAt":     llx.TimeDataPtr(roleAssignment.Properties.CreatedOn),
			"updatedAt":     llx.TimeDataPtr(roleAssignment.Properties.UpdatedOn),
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
		a.Role.State = plugin.StateIsSet | plugin.StateIsNull
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

func (a *mqlAzureSubscriptionAuthorizationService) managedIdentities() ([]any, error) {
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

	// list all role assignments since we need to attach them to the managed identities
	roleAssignments := a.GetRoleAssignments().Data

	// list user assigned identities
	pager := client.NewListBySubscriptionPager(nil)
	res := []any{}
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
			mqlManagedIdentity.RoleAssignments = plugin.TValue[[]any]{Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

			assignedRoles := []any{}
			for i := range roleAssignments {
				roleAssignment := roleAssignments[i].(*mqlAzureSubscriptionAuthorizationServiceRoleAssignment)
				// Compare the principal ID values, not the whole TValue structs
				// (which also carry State/Error and would only match when those
				// happen to be identical too).
				if roleAssignment.PrincipalId.Data == mqlManagedIdentity.PrincipalId.Data {
					assignedRoles = append(assignedRoles, roleAssignment)
				}
			}

			if len(assignedRoles) > 0 {
				mqlManagedIdentity.RoleAssignments = plugin.TValue[[]any]{Error: nil, Data: assignedRoles, State: plugin.StateIsSet}
			}

			res = append(res, mqlManagedIdentity)
		}
	}
	return res, nil
}

// initAzureSubscriptionManagedIdentity resolves a managed identity that is
// referenced only by its resource ID (e.g. from a typed cross-reference) by
// fetching it on demand. When the identity was already listed by
// managedIdentities(), the runtime cache short-circuits this and the init
// never runs.
func initAzureSubscriptionManagedIdentity(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) != 1 {
		return args, nil, nil
	}
	idRaw, ok := args["__id"]
	if !ok {
		return args, nil, nil
	}
	id, ok := idRaw.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return args, nil, nil
	}
	name, err := resourceID.Component("userAssignedIdentities")
	if err != nil {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	client, err := armmsi.NewUserAssignedIdentitiesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}

	identity, err := client.Get(context.Background(), resourceID.ResourceGroup, name, nil)
	if err != nil {
		// The identity may be inaccessible (deleted, cross-subscription, or
		// access denied); fall back to the bare reference rather than
		// failing the surrounding query.
		return args, nil, nil
	}

	mqlIdentity, err := newMqlManagedIdentity(runtime, &identity.Identity)
	if err != nil {
		return nil, nil, err
	}
	return args, mqlIdentity, nil
}

type mqlAzureSubscriptionManagedIdentityInternal struct {
	cacheResourceID string
	cacheSystemData any
}

func newMqlManagedIdentity(runtime *plugin.Runtime, managedIdentity *armmsi.Identity) (*mqlAzureSubscriptionManagedIdentity, error) {
	// Properties (and the *TenantID inside it) are nullable; guard before
	// dereferencing to avoid a panic on a partially-populated identity.
	var clientID, principalID, tenantID *string
	if p := managedIdentity.Properties; p != nil {
		clientID = p.ClientID
		principalID = p.PrincipalID
		tenantID = (*string)(p.TenantID)
	}
	r, err := CreateResource(runtime, "azure.subscription.managedIdentity",
		map[string]*llx.RawData{
			"__id":        llx.StringDataPtr(managedIdentity.ID),
			"name":        llx.StringDataPtr(managedIdentity.Name),
			"clientId":    llx.StringDataPtr(clientID),
			"principalId": llx.StringDataPtr(principalID),
			"tenantId":    llx.StringDataPtr(tenantID),
		})
	if err != nil {
		return nil, err
	}

	mqlManagedIdentity := r.(*mqlAzureSubscriptionManagedIdentity)
	mqlManagedIdentity.cacheResourceID = convert.ToValue(managedIdentity.ID)

	sysData, err := convert.JsonToDict(managedIdentity.SystemData)
	if err != nil {
		return nil, err
	}
	mqlManagedIdentity.cacheSystemData = sysData

	return mqlManagedIdentity, nil
}

func (a *mqlAzureSubscriptionManagedIdentity) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.__id, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionManagedIdentity) roleAssignments() ([]any, error) {
	// NOTE: this should never be called since we assign roles during the managed identities query
	return nil, errors.New("could not fetch role assignments for managed identities")
}

func (a *mqlAzureSubscriptionManagedIdentity) federatedIdentityCredentials() ([]any, error) {
	if a.cacheResourceID == "" {
		return []any{}, nil
	}
	resourceID, err := ParseResourceID(a.cacheResourceID)
	if err != nil {
		return nil, err
	}
	name, err := resourceID.Component("userAssignedIdentities")
	if err != nil {
		return nil, err
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	client, err := armmsi.NewFederatedIdentityCredentialsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListPager(resourceID.ResourceGroup, name, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, fic := range page.Value {
			if fic == nil {
				continue
			}
			entry := map[string]any{
				"name": convert.ToValue(fic.Name),
			}
			if p := fic.Properties; p != nil {
				entry["issuer"] = convert.ToValue(p.Issuer)
				entry["subject"] = convert.ToValue(p.Subject)
				audiences := []any{}
				for _, aud := range p.Audiences {
					if aud != nil {
						audiences = append(audiences, *aud)
					}
				}
				entry["audiences"] = audiences
			}
			res = append(res, entry)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceDenyAssignment) id() (string, error) {
	return a.Id.Data, nil
}

// denyAssignments lists subscription-scoped read-only deny rules. Includes managed-app deny
// assignments and any custom deny rules. Required for "no Owner can delete X" verification.
func (a *mqlAzureSubscriptionAuthorizationService) denyAssignments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	client, err := authorization.NewDenyAssignmentsClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, da := range page.Value {
			if da == nil {
				continue
			}
			var denyName, description, scope string
			var doNotApplyToChild, isSystemProtected *bool
			var perms, principals, excludePrincipals []any
			if p := da.Properties; p != nil {
				if p.DenyAssignmentName != nil {
					denyName = *p.DenyAssignmentName
				}
				if p.Description != nil {
					description = *p.Description
				}
				if p.Scope != nil {
					scope = *p.Scope
				}
				doNotApplyToChild = p.DoNotApplyToChildScopes
				isSystemProtected = p.IsSystemProtected
				for _, pm := range p.Permissions {
					if pm == nil {
						continue
					}
					if d, err := convert.JsonToDict(pm); err == nil {
						perms = append(perms, d)
					}
				}
				for _, pr := range p.Principals {
					if pr == nil {
						continue
					}
					if d, err := convert.JsonToDict(pr); err == nil {
						principals = append(principals, d)
					}
				}
				for _, pr := range p.ExcludePrincipals {
					if pr == nil {
						continue
					}
					if d, err := convert.JsonToDict(pr); err == nil {
						excludePrincipals = append(excludePrincipals, d)
					}
				}
			}

			mqlDa, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.denyAssignment",
				map[string]*llx.RawData{
					"id":                      llx.StringDataPtr(da.ID),
					"name":                    llx.StringDataPtr(da.Name),
					"type":                    llx.StringDataPtr(da.Type),
					"denyAssignmentName":      llx.StringData(denyName),
					"description":             llx.StringData(description),
					"doNotApplyToChildScopes": llx.BoolDataPtr(doNotApplyToChild),
					"isSystemProtected":       llx.BoolDataPtr(isSystemProtected),
					"scope":                   llx.StringData(scope),
					"permissions":             llx.ArrayData(perms, types.Dict),
					"principals":              llx.ArrayData(principals, types.Dict),
					"excludePrincipals":       llx.ArrayData(excludePrincipals, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDa)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAuthorizationServiceClassicAdministrator) id() (string, error) {
	return a.Id.Data, nil
}

// classicAdministrators lists legacy ASM co-admins / service admins on the subscription.
// CIS 1.21 requires this to be empty. Distinct from RBAC role assignments.
func (a *mqlAzureSubscriptionAuthorizationService) classicAdministrators() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	client, err := authorization.NewClassicAdministratorsClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ca := range page.Value {
			if ca == nil {
				continue
			}
			var emailAddress, role string
			if ca.Properties != nil {
				if ca.Properties.EmailAddress != nil {
					emailAddress = *ca.Properties.EmailAddress
				}
				if ca.Properties.Role != nil {
					role = *ca.Properties.Role
				}
			}

			mqlCa, err := CreateResource(a.MqlRuntime, "azure.subscription.authorizationService.classicAdministrator",
				map[string]*llx.RawData{
					"id":           llx.StringDataPtr(ca.ID),
					"name":         llx.StringDataPtr(ca.Name),
					"type":         llx.StringDataPtr(ca.Type),
					"emailAddress": llx.StringData(emailAddress),
					"role":         llx.StringData(role),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCa)
		}
	}
	return res, nil
}

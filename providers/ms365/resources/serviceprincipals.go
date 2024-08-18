// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/stringx"
)

const (
	// Microsoft Entra Tenant IDs for first party apps as defined in
	// https://learn.microsoft.com/en-us/troubleshoot/azure/entra/entra-id/governance/verify-first-party-apps-sign-in
	MicrosoftEntraTenantID = "f8cdef31-a31e-4b4a-93e4-5f571e91255a"
	MicrosoftTenantID      = "72f988bf-86f1-41af-91ab-2d7cd011db47"
)

// index for service principal app roles
var idxAppRoleMutex = &sync.RWMutex{}
var idxOauthScopeMutex = &sync.RWMutex{}
var idxAppNamesMutex = &sync.RWMutex{}

type roleCache struct {
	id         string
	desc       string
	permission string
}

type oauth2PermissionScope struct {
	id         string
	desc       string
	permission string
}

type PermissionIndexer interface {
	IndexAppName(spId, appName string)
	IndexAppRole(spId string, roles ...models.AppRoleable)
	IndexOauthScope(spId string, scopes ...models.PermissionScopeable)
}

// creates a new permission indexer to cache the service principal app roles and oauth scopes
type permissionIndexer struct {
	// permission index key : appId/roleID value: permission
	idxAppRoles map[string]*roleCache
	// permission index key : appId/permission value: permission
	idxOauthPermissionScopes map[string]*oauth2PermissionScope
	// index of app names by appId
	idxAppNames map[string]string
}

func (idx *permissionIndexer) initAppRoleIndex() {
	if idx.idxAppRoles == nil {
		idx.idxAppRoles = make(map[string]*roleCache)
	}
}

func (idx *permissionIndexer) initOauthPermissionScopeIndex() {
	if idx.idxOauthPermissionScopes == nil {
		idx.idxOauthPermissionScopes = make(map[string]*oauth2PermissionScope)
	}
}

func (idx *permissionIndexer) initAppNameIndex() {
	if idx.idxAppNames == nil {
		idx.idxAppNames = make(map[string]string)
	}
}

func (idx *permissionIndexer) IndexAppName(spId, name string) {
	idx.initAppNameIndex()
	idxAppNamesMutex.Lock()
	idx.idxAppNames[spId] = name
	idxAppNamesMutex.Unlock()
}

func (idx *permissionIndexer) appName(appId string) (string, bool) {
	if idx.idxAppNames == nil {
		return "", false
	}
	idxAppNamesMutex.RLock()
	name, ok := idx.idxAppNames[appId]
	idxAppNamesMutex.RUnlock()
	return name, ok
}

// index all app roles we've seen for service-accounts
func (idx *permissionIndexer) IndexAppRole(spId string, roles ...models.AppRoleable) {
	idx.initAppRoleIndex()
	idxAppRoleMutex.Lock()
	for _, r := range roles {
		roleId := r.GetId()
		if roleId == nil || roleId.String() == "" {
			continue
		}

		rCache := &roleCache{
			id:         roleId.String(),
			desc:       convert.ToString(r.GetDisplayName()),
			permission: convert.ToString(r.GetValue()),
		}

		idx.idxAppRoles[spId+"/"+rCache.id] = rCache
		idx.idxAppRoles[spId+"/"+rCache.permission] = rCache
	}
	idxAppRoleMutex.Unlock()
}

// appRole returns a role by appId and roleID
func (idx *permissionIndexer) appRole(spId, roleId string) (*roleCache, bool) {
	if idx.idxAppRoles == nil {
		return nil, false
	}
	idxAppRoleMutex.RLock()
	role, ok := idx.idxAppRoles[spId+"/"+roleId]
	idxAppRoleMutex.RUnlock()
	return role, ok
}

// index all oauth scopes we've seen for service-accounts
func (idx *permissionIndexer) IndexOauthScope(spId string, scopes ...models.PermissionScopeable) {
	idxOauthScopeMutex.Lock()
	idx.initOauthPermissionScopeIndex()
	for _, s := range scopes {
		scopeId := s.GetId()
		if scopeId == nil || scopeId.String() == "" {
			continue
		}

		scope := &oauth2PermissionScope{
			id:         scopeId.String(),
			desc:       convert.ToString(s.GetAdminConsentDisplayName()),
			permission: convert.ToString(s.GetValue()),
		}

		idx.idxOauthPermissionScopes[spId+"/"+scope.permission] = scope
	}
	idxOauthScopeMutex.Unlock()
}

func (idx *permissionIndexer) getOauthPermissionScope(appId, permission string) (*oauth2PermissionScope, bool) {
	if idx.idxOauthPermissionScopes == nil {
		return nil, false
	}
	idxOauthScopeMutex.RLock()
	scope, ok := idx.idxOauthPermissionScopes[appId+"/"+permission]
	idxOauthScopeMutex.RUnlock()
	return scope, ok
}

var servicePrincipalFields = []string{
	"id",
	"servicePrincipalType",
	"displayName",
	"appId",
	"appOwnerOrganizationId",
	"description",
	"tags",
	"accountEnabled",
	"homepage",
	"replyUrls",
	"appRoleAssignmentRequired",
	"notes",
	"applicationTemplateId",
	"loginUrl",
	"logoutUrl",
	"servicePrincipalNames",
	"signInAudience",
	"preferredSingleSignOnMode",
	"notificationEmailAddresses",
	"appRoleAssignmentRequired",
	"accountEnabled",
	"verifiedPublisher",
	"appRoles",
	"oauth2PermissionScopes",
}

func (m *mqlMicrosoftServiceprincipal) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftServiceprincipalAssignment) id() (string, error) {
	return m.Id.Data, nil
}

func initMicrosoftServiceprincipal(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the service principal if we have been supplied by its name and nothing else
	rawName, okName := args["name"]
	rawId, okId := args["id"]
	rawAppId, okAppId := args["appId"]

	if len(args) != 1 || (!okName && !okId && !okAppId) {
		return args, nil, nil
	}

	var filter func(sp *mqlMicrosoftServiceprincipal) bool

	if okId {
		id := rawId.Value.(string)
		filter = func(sp *mqlMicrosoftServiceprincipal) bool {
			return sp.Id.Data == id
		}
	} else if okAppId {
		appId := rawAppId.Value.(string)
		filter = func(sp *mqlMicrosoftServiceprincipal) bool {
			return sp.AppId.Data == appId
		}
	} else if okName {
		// NOTE: be aware that service principal names are not unique
		name := rawName.Value.(string)
		filter = func(sp *mqlMicrosoftServiceprincipal) bool {
			return sp.Name.Data == name
		}
	}

	if filter == nil {
		return nil, nil, errors.New("invalid filter")
	}

	mqlResource, err := runtime.CreateResource(runtime, "microsoft", nil)
	if err != nil {
		return nil, nil, err
	}
	microsoftResource := mqlResource.(*mqlMicrosoft)
	servicePrincipalList := microsoftResource.GetServiceprincipals()
	for i := range servicePrincipalList.Data {
		sp := servicePrincipalList.Data[i].(*mqlMicrosoftServiceprincipal)
		if filter(sp) {
			return nil, sp, nil
		}
	}

	return nil, nil, errors.New("service principal not found")
}

// enterprise applications are just service principals with a special tag, attached to them
// this is the same way the portal UI fetches the enterprise apps by looking for the tag
func (a *mqlMicrosoft) enterpriseApplications() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	top := int32(999)
	filter := "tags/Any(x: x eq 'WindowsAzureActiveDirectoryIntegratedApp')"
	params := &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
		Top:    &top,
		Filter: &filter,
		Expand: []string{"appRoleAssignedTo"},
	}
	return fetchServicePrincipals(a.MqlRuntime, conn, params, nil)
}

func (a *mqlMicrosoft) serviceprincipals() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	top := int32(999)
	params := &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
		Top:    &top,
		Select: servicePrincipalFields,
	}

	return fetchServicePrincipals(a.MqlRuntime, conn, params, a)
}

func fetchServicePrincipals(runtime *plugin.Runtime, conn *connection.Ms365Connection, params *serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters, permissionIndexer PermissionIndexer) ([]interface{}, error) {
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: params,
	})
	if err != nil {
		return nil, transformError(err)
	}
	sps, err := iterate[*models.ServicePrincipal](ctx, resp, graphClient.GetAdapter(), serviceprincipals.CreateDeltaGetResponseFromDiscriminatorValue)
	if err != nil {
		return nil, transformError(err)
	}
	res := []interface{}{}
	for _, sp := range sps {
		// create resource
		mqlResource, err := newMqlMicrosoftServicePrincipal(runtime, sp)
		if err != nil {
			return nil, err
		}

		// also fill up the index
		if permissionIndexer != nil {
			permissionIndexer.IndexAppName(convert.ToString(sp.GetId()), *sp.GetDisplayName())
			permissionIndexer.IndexAppRole(convert.ToString(sp.GetId()), sp.GetAppRoles()...)
			permissionIndexer.IndexOauthScope(convert.ToString(sp.GetId()), sp.GetOauth2PermissionScopes()...)
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func newMqlMicrosoftServicePrincipal(runtime *plugin.Runtime, sp models.ServicePrincipalable) (*mqlMicrosoftServiceprincipal, error) {
	hideApp := stringx.Contains(sp.GetTags(), "HideApp")
	assignments := []interface{}{}
	for _, a := range sp.GetAppRoleAssignedTo() {
		assignment, err := CreateResource(runtime, "microsoft.serviceprincipal.assignment", map[string]*llx.RawData{
			"id":          llx.StringDataPtr(a.GetId()),
			"displayName": llx.StringDataPtr(a.GetPrincipalDisplayName()),
			"type":        llx.StringDataPtr(a.GetPrincipalType()),
		})
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	var appVerifiedOrganizationID string
	if sp.GetAppOwnerOrganizationId() != nil {
		appVerifiedOrganizationID = sp.GetAppOwnerOrganizationId().String()
	}

	verifiedPublisher, _ := convert.JsonToDict(newVerifiedPublisher(sp.GetVerifiedPublisher()))

	mqlAppRoleList := []interface{}{}
	appRoles := sp.GetAppRoles()
	for i := range appRoles {
		appRole := appRoles[i]

		uuid := appRole.GetId()
		if uuid == nil {
			log.Debug().Msg("appRole ID is nil")
			continue
		}

		mqlAppRoleResource, err := CreateResource(runtime, "microsoft.application.role",
			map[string]*llx.RawData{
				"__id":               llx.StringData(uuid.String()),
				"id":                 llx.StringData(uuid.String()),
				"name":               llx.StringDataPtr(appRole.GetDisplayName()),
				"description":        llx.StringDataPtr(appRole.GetDescription()),
				"value":              llx.StringDataPtr(appRole.GetValue()),
				"allowedMemberTypes": llx.ArrayData(convert.SliceAnyToInterface(appRole.GetAllowedMemberTypes()), types.String),
				"isEnabled":          llx.BoolDataPtr(appRole.GetIsEnabled()),
			})
		if err != nil {
			return nil, err
		}
		mqlAppRoleList = append(mqlAppRoleList, mqlAppRoleResource)
	}

	args := map[string]*llx.RawData{
		"id":                         llx.StringDataPtr(sp.GetId()),
		"type":                       llx.StringDataPtr(sp.GetServicePrincipalType()),
		"name":                       llx.StringDataPtr(sp.GetDisplayName()),
		"appId":                      llx.StringDataPtr(sp.GetAppId()),
		"appOwnerOrganizationId":     llx.StringData(appVerifiedOrganizationID),
		"description":                llx.StringDataPtr(sp.GetDescription()),
		"tags":                       llx.ArrayData(convert.SliceAnyToInterface(sp.GetTags()), types.String),
		"enabled":                    llx.BoolDataPtr(sp.GetAccountEnabled()),
		"homepageUrl":                llx.StringDataPtr(sp.GetHomepage()),
		"replyUrls":                  llx.ArrayData(convert.SliceAnyToInterface(sp.GetReplyUrls()), types.String),
		"assignmentRequired":         llx.BoolDataPtr(sp.GetAppRoleAssignmentRequired()),
		"visibleToUsers":             llx.BoolData(!hideApp),
		"notes":                      llx.StringDataPtr(sp.GetNotes()),
		"assignments":                llx.ArrayData(assignments, types.ResourceLike),
		"applicationTemplateId":      llx.StringDataPtr(sp.GetApplicationTemplateId()),
		"loginUrl":                   llx.StringDataPtr(sp.GetLoginUrl()),
		"logoutUrl":                  llx.StringDataPtr(sp.GetLogoutUrl()),
		"servicePrincipalNames":      llx.ArrayData(convert.SliceAnyToInterface(sp.GetServicePrincipalNames()), types.String),
		"signInAudience":             llx.StringDataPtr(sp.GetSignInAudience()),
		"preferredSingleSignOnMode":  llx.StringDataPtr(sp.GetPreferredSingleSignOnMode()),
		"notificationEmailAddresses": llx.ArrayData(convert.SliceAnyToInterface(sp.GetNotificationEmailAddresses()), types.String),
		"appRoleAssignmentRequired":  llx.BoolDataPtr(sp.GetAppRoleAssignmentRequired()),
		"accountEnabled":             llx.BoolDataPtr(sp.GetAccountEnabled()),
		"verifiedPublisher":          llx.DictData(verifiedPublisher),
		"appRoles":                   llx.ArrayData(mqlAppRoleList, types.Resource("microsoft.application.role")),
	}
	info := sp.GetInfo()
	if info != nil {
		args["termsOfServiceUrl"] = llx.StringDataPtr(info.GetTermsOfServiceUrl())
	}
	mqlResource, err := CreateResource(runtime, "microsoft.serviceprincipal", args)
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftServiceprincipal), nil
}

func (a *mqlMicrosoftServiceprincipal) isFirstParty() (bool, error) {
	ownerId := a.AppOwnerOrganizationId.Data
	// e.g. O365 LinkedIn Connection and YammerOnOls do not have an owner
	if ownerId == MicrosoftEntraTenantID || ownerId == MicrosoftTenantID || ownerId == "" {
		return true, nil
	}
	return false, nil
}

func (a *mqlMicrosoftServiceprincipal) permissions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	servicePrincipalId := a.Id.Data

	res, err := CreateResource(a.MqlRuntime, "microsoft", nil)
	if err != nil {
		return nil, err
	}
	mqlMicrosoftResource := res.(*mqlMicrosoft)

	// fetch service credentials to build up the app role index
	mqlMicrosoftResource.GetServiceprincipals()

	// fetch all role assignments for the service principal, those are "application" types
	ctx := context.Background()
	grantedApplicationRolesResp, err := graphClient.ServicePrincipals().ByServicePrincipalId(servicePrincipalId).AppRoleAssignments().Get(ctx, &serviceprincipals.ItemAppRoleAssignmentsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	// TODO: also list ungranted app roles
	list := []interface{}{}
	appRolesAssignments := grantedApplicationRolesResp.GetValue()
	for _, roleAssignment := range appRolesAssignments {
		assignmentID := roleAssignment.GetId()
		spId := roleAssignment.GetResourceId()  // id of the service account
		roleId := roleAssignment.GetAppRoleId() // id of the app role on the service account

		if assignmentID == nil || spId == nil || roleId == nil {
			continue
		}
		role, ok := mqlMicrosoftResource.appRole(spId.String(), roleId.String())
		if !ok {
			log.Debug().Msgf("role not found in cache: %v", roleId)
			continue
		}

		assignment, err := CreateResource(a.MqlRuntime, "microsoft.application.permission", map[string]*llx.RawData{
			"__id":        llx.StringDataPtr(assignmentID),
			"appId":       llx.StringData(spId.String()),
			"appName":     llx.StringDataPtr(roleAssignment.GetResourceDisplayName()),
			"description": llx.StringData(role.desc),
			"id":          llx.StringData(roleId.String()),
			"name":        llx.StringData(role.permission),
			"type":        llx.StringData("application"),
			"status":      llx.StringData("granted"),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, assignment)
	}

	oauthResp, err := graphClient.ServicePrincipals().ByServicePrincipalId(servicePrincipalId).Oauth2PermissionGrants().Get(ctx, &serviceprincipals.ItemOauth2PermissionGrantsRequestBuilderGetRequestConfiguration{})
	delegatedRolesAssignments := oauthResp.GetValue()
	for _, roleAssignment := range delegatedRolesAssignments {

		spId := roleAssignment.GetResourceId() // id of the service account
		if spId == nil {
			continue
		}

		appName, _ := mqlMicrosoftResource.appName(*spId)
		scope := roleAssignment.GetScope()
		if scope == nil {
			continue
		}

		// one line can include multiple scopes
		scopeList := strings.Split(*scope, " ")

		for _, scopeEntry := range scopeList {
			if scopeEntry == "" {
				continue
			}
			id := convert.ToString(roleAssignment.GetId())
			desc := ""
			role, ok := mqlMicrosoftResource.getOauthPermissionScope(*spId, scopeEntry)
			if ok {
				desc = role.desc
			}

			assignment, err := CreateResource(a.MqlRuntime, "microsoft.application.permission", map[string]*llx.RawData{
				"__id":        llx.StringData(id + "/" + scopeEntry),
				"appId":       llx.StringDataPtr(spId),
				"appName":     llx.StringData(appName),
				"description": llx.StringData(desc),
				"id":          llx.StringData(id),
				"name":        llx.StringData(scopeEntry),
				"type":        llx.StringData("delegated"),
				"status":      llx.StringData("granted"),
			})
			if err != nil {
				return nil, err
			}
			list = append(list, assignment)
		}
	}
	return list, nil
}

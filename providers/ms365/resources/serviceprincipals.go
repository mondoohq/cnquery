// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"github.com/rs/zerolog/log"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
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

func (m *mqlMicrosoftServiceprincipal) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftServiceprincipalAssignment) id() (string, error) {
	return m.Id.Data, nil
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
	return fetchServicePrincipals(a.MqlRuntime, conn, params)
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
}

func (a *mqlMicrosoft) serviceprincipals() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	top := int32(999)
	params := &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
		Top:    &top,
		Select: servicePrincipalFields,
	}
	return fetchServicePrincipals(a.MqlRuntime, conn, params)
}

func fetchServicePrincipals(runtime *plugin.Runtime, conn *connection.Ms365Connection, params *serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters) ([]interface{}, error) {
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
		mqlResource, err := newMqlMicrosoftServicePrincipal(runtime, sp)
		if err != nil {
			return nil, err
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

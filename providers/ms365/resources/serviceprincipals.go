// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipals"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/cnquery/v10/utils/stringx"
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

func (a *mqlMicrosoft) serviceprincipals() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	top := int32(999)
	params := &serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters{
		Top: &top,
	}
	return fetchServicePrincipals(a.MqlRuntime, conn, params)
}

func fetchServicePrincipals(runtime *plugin.Runtime, conn *connection.Ms365Connection, params *serviceprincipals.ServicePrincipalsRequestBuilderGetQueryParameters) ([]interface{}, error) {
	graphClient, err := graphClient(conn)
	if err != nil {
		return nil, err
	} // TODO: what if we have more than 1k SPs?
	ctx := context.Background()
	resp, err := graphClient.ServicePrincipals().Get(ctx, &serviceprincipals.ServicePrincipalsRequestBuilderGetRequestConfiguration{
		QueryParameters: params,
	})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	sps := resp.GetValue()
	for _, sp := range sps {
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
		args := map[string]*llx.RawData{
			"id":                 llx.StringDataPtr(sp.GetId()),
			"name":               llx.StringDataPtr(sp.GetDisplayName()),
			"type":               llx.StringDataPtr(sp.GetServicePrincipalType()),
			"tags":               llx.ArrayData(convert.SliceAnyToInterface(sp.GetTags()), types.String),
			"enabled":            llx.BoolDataPtr(sp.GetAccountEnabled()),
			"homepageUrl":        llx.StringDataPtr(sp.GetHomepage()),
			"replyUrls":          llx.ArrayData(convert.SliceAnyToInterface(sp.GetReplyUrls()), types.String),
			"assignmentRequired": llx.BoolDataPtr(sp.GetAppRoleAssignmentRequired()),
			"visibleToUsers":     llx.BoolData(!hideApp),
			"notes":              llx.StringDataPtr(sp.GetNotes()),
			"assignments":        llx.ArrayData(assignments, types.ResourceLike),
		}
		info := sp.GetInfo()
		if info != nil {
			args["termsOfServiceUrl"] = llx.StringDataPtr(info.GetTermsOfServiceUrl())
		}
		mqlResource, err := CreateResource(runtime, "microsoft.serviceprincipal", args)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

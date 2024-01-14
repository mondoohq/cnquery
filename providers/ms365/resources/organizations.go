// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (m *mqlMicrosoftOrganization) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoft) organizations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := graphClient(conn)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Organization().Get(ctx, &organization.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	orgs := resp.GetValue()
	for i := range orgs {
		org := orgs[i]

		assignedPlans, err := convert.JsonToDictSlice(newAssignedPlans(org.GetAssignedPlans()))
		if err != nil {
			return nil, err
		}
		verifiedDomains, err := convert.JsonToDictSlice(newVerifiedDomains(org.GetVerifiedDomains()))
		if err != nil {
			return nil, err
		}
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.organization",
			map[string]*llx.RawData{
				"id":                    llx.StringData(convert.ToString(org.GetId())),
				"assignedPlans":         llx.ArrayData(assignedPlans, types.Any),
				"createdDateTime":       llx.TimeDataPtr(org.GetCreatedDateTime()),
				"displayName":           llx.StringData(convert.ToString(org.GetDisplayName())),
				"verifiedDomains":       llx.ArrayData(verifiedDomains, types.Any),
				"onPremisesSyncEnabled": llx.BoolDataPtr(org.GetOnPremisesSyncEnabled()),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

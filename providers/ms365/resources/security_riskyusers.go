// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"github.com/microsoftgraph/msgraph-sdk-go/identityprotection"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
)

// riskyUsers returns a list of risky users
// requires IdentityRiskyUser.Read.All permission
// see https://learn.microsoft.com/en-us/graph/api/resources/riskyuser?view=graph-rest-1.0
func (a *mqlMicrosoftSecurity) riskyUsers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	filter := "riskState eq 'atRisk'"
	resp, err := graphClient.IdentityProtection().RiskyUsers().Get(ctx, &identityprotection.RiskyUsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &identityprotection.RiskyUsersRequestBuilderGetQueryParameters{
			Filter: &filter,
		},
	})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	users := resp.GetValue()
	for i := range users {
		riskyUser := users[i]
		mqlResource, err := newMqlMicrosoftRiskyUser(a.MqlRuntime, riskyUser)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlResource)
	}
	return res, nil
}

func newMqlMicrosoftRiskyUser(runtime *plugin.Runtime, riskyUser models.RiskyUserable) (*mqlMicrosoftSecurityRiskyUser, error) {
	if riskyUser == nil {
		return nil, nil
	}

	var detail *string
	if riskyUser.GetRiskDetail() != nil {
		detailData := riskyUser.GetRiskDetail().String()
		detail = &detailData
	}

	var riskLevel *string
	if riskyUser.GetRiskLevel() != nil {
		riskLevelData := riskyUser.GetRiskLevel().String()
		riskLevel = &riskLevelData
	}

	var riskState *string
	if riskyUser.GetRiskState() != nil {
		riskStateData := riskyUser.GetRiskState().String()
		riskState = &riskStateData
	}

	mqlResource, err := CreateResource(runtime, "microsoft.security.riskyUser",
		map[string]*llx.RawData{
			"__id":          llx.StringDataPtr(riskyUser.GetId()),
			"id":            llx.StringDataPtr(riskyUser.GetId()),
			"name":          llx.StringDataPtr(riskyUser.GetUserDisplayName()),
			"principalName": llx.StringDataPtr(riskyUser.GetUserPrincipalName()),
			"riskDetail":    llx.StringDataPtr(detail),
			"riskLevel":     llx.StringDataPtr(riskLevel),
			"riskState":     llx.StringDataPtr(riskState),
			"lastUpdatedAt": llx.TimeDataPtr(riskyUser.GetRiskLastUpdatedDateTime()),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftSecurityRiskyUser), nil
}

func (r *mqlMicrosoftSecurityRiskyUser) user() (*mqlMicrosoftUser, error) {
	user, err := NewResource(r.MqlRuntime, "microsoft.user", map[string]*llx.RawData{
		"id": llx.StringData(r.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return user.(*mqlMicrosoftUser), nil
}

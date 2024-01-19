// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/security"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (m *mqlMicrosoftSecuritySecurityscore) id() (string, error) {
	return m.Id.Data, nil
}

func msSecureScoreToMql(runtime *plugin.Runtime, score models.SecureScoreable) (*mqlMicrosoftSecuritySecurityscore, error) {
	if score == nil {
		return nil, nil
	}
	averageComparativeScores := []interface{}{}
	graphAverageComparativeScores := score.GetAverageComparativeScores()
	for j := range graphAverageComparativeScores {
		entry, err := convert.JsonToDict(newAverageComparativeScore(graphAverageComparativeScores[j]))
		if err != nil {
			return nil, err
		}
		averageComparativeScores = append(averageComparativeScores, entry)
	}

	controlScores := []interface{}{}
	graphControlScores := score.GetControlScores()
	for j := range graphControlScores {
		entry, err := convert.JsonToDict(newControlScore(graphControlScores[j]))
		if err != nil {
			return nil, err
		}
		controlScores = append(controlScores, entry)
	}

	vendorInformation, err := convert.JsonToDict(newSecurityVendorInformation(score.GetVendorInformation()))
	if err != nil {
		return nil, err
	}

	enabledServices := []interface{}{}
	for _, service := range score.GetEnabledServices() {
		enabledServices = append(enabledServices, service)
	}
	mqlResource, err := CreateResource(runtime, "microsoft.security.securityscore",
		map[string]*llx.RawData{
			"id":                       llx.StringData(convert.ToString(score.GetId())),
			"activeUserCount":          llx.IntData(convert.ToInt64From32(score.GetActiveUserCount())),
			"averageComparativeScores": llx.ArrayData(averageComparativeScores, types.Any),
			"azureTenantId":            llx.StringData(convert.ToString(score.GetAzureTenantId())),
			"controlScores":            llx.ArrayData(controlScores, types.Any),
			"createdDateTime":          llx.TimeDataPtr(score.GetCreatedDateTime()),
			"currentScore":             llx.FloatData(convert.ToFloat64(score.GetCurrentScore())),
			"enabledServices":          llx.ArrayData(enabledServices, types.String),
			"licensedUserCount":        llx.IntData(convert.ToInt64From32(score.GetLicensedUserCount())),
			"maxScore":                 llx.FloatData(convert.ToFloat64(score.GetMaxScore())),
			"vendorInformation":        llx.DictData(vendorInformation),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftSecuritySecurityscore), nil
}

func (a *mqlMicrosoftSecurity) latestSecureScores() (*mqlMicrosoftSecuritySecurityscore, error) {
	secureScores := a.GetSecureScores()
	if secureScores.Error != nil {
		return nil, secureScores.Error
	}
	if len(secureScores.Data) == 0 {
		return nil, errors.New("could not retrieve any score")
	}

	latest := secureScores.Data[0].(*mqlMicrosoftSecuritySecurityscore)
	for _, s := range secureScores.Data {
		mqlS := s.(*mqlMicrosoftSecuritySecurityscore)
		if mqlS.CreatedDateTime.Data.After(*latest.CreatedDateTime.Data) {
			latest = mqlS
		}
	}
	return latest, nil
}

// see https://docs.microsoft.com/en-us/graph/api/securescore-get?view=graph-rest-1.0&tabs=http
func (a *mqlMicrosoftSecurity) secureScores() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := graphClient(conn)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Security().SecureScores().Get(ctx, &security.SecureScoresRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	scores := resp.GetValue()
	for i := range scores {
		score := scores[i]
		mqlResource, err := msSecureScoreToMql(a.MqlRuntime, score)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlResource)
	}
	return res, nil
}

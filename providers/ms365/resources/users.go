// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (m *mqlMicrosoftUser) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoft) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	selectFields := []string{
		"id", "accountEnabled", "city", "companyName", "country", "createdDateTime", "department", "displayName", "employeeId", "givenName",
		"jobTitle", "mail", "mobilePhone", "otherMails", "officeLocation", "postalCode", "state", "streetAddress", "surname", "userPrincipalName", "userType",
	}
	ctx := context.Background()
	resp, err := graphClient.Users().Get(ctx, &users.UsersRequestBuilderGetRequestConfiguration{QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
		Select: selectFields,
	}})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	users := resp.GetValue()
	for _, u := range users {
		graphUser, err := CreateResource(a.MqlRuntime, "microsoft.user",
			map[string]*llx.RawData{
				"id":                llx.StringDataPtr(u.GetId()),
				"accountEnabled":    llx.BoolDataPtr(u.GetAccountEnabled()),
				"city":              llx.StringDataPtr(u.GetCity()),
				"companyName":       llx.StringDataPtr(u.GetCompanyName()),
				"country":           llx.StringDataPtr(u.GetCountry()),
				"createdDateTime":   llx.TimeDataPtr(u.GetCreatedDateTime()),
				"department":        llx.StringDataPtr(u.GetDepartment()),
				"displayName":       llx.StringDataPtr(u.GetDisplayName()),
				"employeeId":        llx.StringDataPtr(u.GetEmployeeId()),
				"givenName":         llx.StringDataPtr(u.GetGivenName()),
				"jobTitle":          llx.StringDataPtr(u.GetJobTitle()),
				"mail":              llx.StringDataPtr(u.GetMail()),
				"mobilePhone":       llx.StringDataPtr(u.GetMobilePhone()),
				"otherMails":        llx.ArrayData(llx.TArr2Raw(u.GetOtherMails()), types.String),
				"officeLocation":    llx.StringDataPtr(u.GetOfficeLocation()),
				"postalCode":        llx.StringDataPtr(u.GetPostalCode()),
				"state":             llx.StringDataPtr(u.GetState()),
				"streetAddress":     llx.StringDataPtr(u.GetStreetAddress()),
				"surname":           llx.StringDataPtr(u.GetSurname()),
				"userPrincipalName": llx.StringDataPtr(u.GetUserPrincipalName()),
				"userType":          llx.StringDataPtr(u.GetUserType()),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, graphUser)
	}

	return res, nil
}

func (a *mqlMicrosoftUser) settings() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	id := a.Id.Data
	userSettings, err := graphClient.Users().ByUserId(id).Settings().Get(ctx, &users.ItemSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	return convert.JsonToDict(newUserSettings(userSettings))
}

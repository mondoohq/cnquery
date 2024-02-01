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
		mails := []interface{}{}
		for _, mail := range u.GetOtherMails() {
			mails = append(mails, mail)
		}
		graphUser, err := CreateResource(a.MqlRuntime, "microsoft.user",
			map[string]*llx.RawData{
				"id":                llx.StringData(convert.ToString(u.GetId())),
				"accountEnabled":    llx.BoolData(convert.ToBool(u.GetAccountEnabled())),
				"city":              llx.StringData(convert.ToString(u.GetCity())),
				"companyName":       llx.StringData(convert.ToString(u.GetCompanyName())),
				"country":           llx.StringData(convert.ToString(u.GetCountry())),
				"createdDateTime":   llx.TimeDataPtr(u.GetCreatedDateTime()),
				"department":        llx.StringData(convert.ToString(u.GetDepartment())),
				"displayName":       llx.StringData(convert.ToString(u.GetDisplayName())),
				"employeeId":        llx.StringData(convert.ToString(u.GetEmployeeId())),
				"givenName":         llx.StringData(convert.ToString(u.GetGivenName())),
				"jobTitle":          llx.StringData(convert.ToString(u.GetJobTitle())),
				"mail":              llx.StringData(convert.ToString(u.GetMail())),
				"mobilePhone":       llx.StringData(convert.ToString(u.GetMobilePhone())),
				"otherMails":        llx.ArrayData(mails, types.String),
				"officeLocation":    llx.StringData(convert.ToString(u.GetOfficeLocation())),
				"postalCode":        llx.StringData(convert.ToString(u.GetPostalCode())),
				"state":             llx.StringData(convert.ToString(u.GetState())),
				"streetAddress":     llx.StringData(convert.ToString(u.GetStreetAddress())),
				"surname":           llx.StringData(convert.ToString(u.GetSurname())),
				"userPrincipalName": llx.StringData(convert.ToString(u.GetUserPrincipalName())),
				"userType":          llx.StringData(convert.ToString(u.GetUserType())),
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

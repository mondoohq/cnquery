// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

var userSelectFields = []string{
	"id", "accountEnabled", "city", "companyName", "country", "createdDateTime", "department", "displayName", "employeeId", "givenName",
	"jobTitle", "mail", "mobilePhone", "otherMails", "officeLocation", "postalCode", "state", "streetAddress", "surname", "userPrincipalName", "userType",
}

func (a *mqlMicrosoft) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	// fetch user data
	ctx := context.Background()
	top := int32(999)
	resp, err := graphClient.Users().Get(
		ctx, &users.UsersRequestBuilderGetRequestConfiguration{
			QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
				Select: userSelectFields,
				Top:    &top,
			},
		},
	)
	if err != nil {
		return nil, transformError(err)
	}
	users, err := iterate[*models.User](ctx, resp, graphClient.GetAdapter(), users.CreateDeltaGetResponseFromDiscriminatorValue)
	if err != nil {
		return nil, transformError(err)
	}

	// construct the result
	res := []interface{}{}
	for _, u := range users {
		graphUser, err := newMqlMicrosoftUser(a.MqlRuntime, u)
		if err != nil {
			return nil, err
		}
		// index users by id and principal name
		a.index(graphUser)
		res = append(res, graphUser)
	}

	return res, nil
}

func initMicrosoftUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the user if we have been supplied by id, displayName or userPrincipalName
	if len(args) > 1 {
		return args, nil, nil
	}

	rawId, okId := args["id"]
	rawDisplayName, okDisplayName := args["displayName"]
	rawPrincipalName, okPrincipalName := args["userPrincipalName"]

	if !okId && !okDisplayName && !okPrincipalName {
		return args, nil, nil
	}

	var filter *string
	if okId {
		idFilter := fmt.Sprintf("id eq '%s'", rawId.Value.(string))
		filter = &idFilter
	} else if okPrincipalName {
		principalNameFilter := fmt.Sprintf("userPrincipalName eq '%s'", rawPrincipalName.Value.(string))
		filter = &principalNameFilter
	} else if okDisplayName {
		displayNameFilter := fmt.Sprintf("displayName eq '%s'", rawDisplayName.Value.(string))
		filter = &displayNameFilter
	}
	if filter == nil {
		return nil, nil, errors.New("no filter found")
	}

	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Users().Get(ctx, &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
			Filter: filter,
		},
	})
	if err != nil {
		return nil, nil, transformError(err)
	}

	val := resp.GetValue()
	if len(val) == 0 {
		return nil, nil, errors.New("user not found")
	}

	userId := val[0].GetId()
	if userId == nil {
		return nil, nil, errors.New("user id not found")
	}

	// fetch user by id
	user, err := graphClient.Users().ByUserId(*userId).Get(ctx, &users.UserItemRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, nil, transformError(err)
	}
	mqlMsApp, err := newMqlMicrosoftUser(runtime, user)
	if err != nil {
		return nil, nil, err
	}

	return nil, mqlMsApp, nil
}

func newMqlMicrosoftUser(runtime *plugin.Runtime, u models.Userable) (*mqlMicrosoftUser, error) {
	graphUser, err := CreateResource(runtime, "microsoft.user",
		map[string]*llx.RawData{
			"__id":              llx.StringDataPtr(u.GetId()),
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
	return graphUser.(*mqlMicrosoftUser), nil
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

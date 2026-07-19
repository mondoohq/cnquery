// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/iam"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlDatabricks) users() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	users, err := acc.Users.ListAll(context.Background(), iam.ListAccountUsersRequest{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range users {
		mqlUser, err := newMqlDatabricksUser(r.MqlRuntime, users[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlUser)
	}
	return list, nil
}

func newMqlDatabricksUser(runtime *plugin.Runtime, user iam.User) (*mqlDatabricksUser, error) {
	emails := []any{}
	for i := range user.Emails {
		if user.Emails[i].Value != "" {
			emails = append(emails, user.Emails[i].Value)
		}
	}

	res, err := CreateResource(runtime, "databricks.user", map[string]*llx.RawData{
		"__id":         llx.StringData("databricks.user/" + user.Id),
		"id":           llx.StringData(user.Id),
		"userName":     llx.StringData(user.UserName),
		"displayName":  llx.StringData(user.DisplayName),
		"active":       llx.BoolData(user.Active),
		"externalId":   llx.StringData(user.ExternalId),
		"emails":       llx.ArrayData(emails, types.String),
		"entitlements": llx.ArrayData(complexValues(user.Entitlements), types.String),
		"roles":        llx.ArrayData(complexValues(user.Roles), types.String),
		"groups":       llx.ArrayData(complexValues(user.Groups), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDatabricksUser), nil
}

func (r *mqlDatabricks) groups() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	groups, err := acc.Groups.ListAll(context.Background(), iam.ListAccountGroupsRequest{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range groups {
		res, err := CreateResource(r.MqlRuntime, "databricks.group", map[string]*llx.RawData{
			"__id":         llx.StringData("databricks.group/" + groups[i].Id),
			"id":           llx.StringData(groups[i].Id),
			"displayName":  llx.StringData(groups[i].DisplayName),
			"externalId":   llx.StringData(groups[i].ExternalId),
			"members":      llx.ArrayData(complexValues(groups[i].Members), types.String),
			"entitlements": llx.ArrayData(complexValues(groups[i].Entitlements), types.String),
			"roles":        llx.ArrayData(complexValues(groups[i].Roles), types.String),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlDatabricks) servicePrincipals() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	sps, err := acc.ServicePrincipals.ListAll(context.Background(), iam.ListAccountServicePrincipalsRequest{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range sps {
		res, err := CreateResource(r.MqlRuntime, "databricks.servicePrincipal", map[string]*llx.RawData{
			"__id":          llx.StringData("databricks.servicePrincipal/" + sps[i].Id),
			"id":            llx.StringData(sps[i].Id),
			"applicationId": llx.StringData(sps[i].ApplicationId),
			"displayName":   llx.StringData(sps[i].DisplayName),
			"active":        llx.BoolData(sps[i].Active),
			"externalId":    llx.StringData(sps[i].ExternalId),
			"entitlements":  llx.ArrayData(complexValues(sps[i].Entitlements), types.String),
			"roles":         llx.ArrayData(complexValues(sps[i].Roles), types.String),
			"groups":        llx.ArrayData(complexValues(sps[i].Groups), types.String),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

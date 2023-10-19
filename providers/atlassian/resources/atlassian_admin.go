// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/admin"
)

func initAtlassianAdminOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn, ok := runtime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, nil, errors.New("Current connection does not allow admin access")
	}
	admin := conn.Client()
	organization, _, err := admin.Organization.Gets(context.Background(), "")
	if err != nil {
		return nil, nil, err
	}

	// We should only ever receive one organization that is scoped to the api key
	// https://community.atlassian.com/t5/Atlassian-Access-questions/Can-we-access-multiple-organisations-using-one-API-Token/qaq-p/1541337
	if len(organization.Data) > 1 {
		return nil, nil, errors.New("Unexpectedly received more than 1 organization")
	}
	org := organization.Data[0]

	args["id"] = llx.StringData(org.ID)
	args["name"] = llx.StringData(org.Attributes.Name)
	args["type"] = llx.StringData(org.Type)

	return args, nil, nil
}

func (a *mqlAtlassianAdminOrganization) managedUsers() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow admin access")
	}

	admin := conn.Client()

	managedUsers, _, err := admin.Organization.Users(context.Background(), a.Id.Data, "")
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, user := range managedUsers.Data {
		mqlAtlassianAdminManagedUser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.managedUser",
			map[string]*llx.RawData{
				"id":         llx.StringData(user.AccountID),
				"name":       llx.StringData(user.Name),
				"type":       llx.StringData(user.AccountType),
				"email":      llx.StringData(user.Email),
				"lastActive": llx.StringData(user.LastActive),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminManagedUser)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationManagedUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganization) policies() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow admin access")
	}
	admin := conn.Client()
	orgId := a.Id.Data
	policies, _, err := admin.Organization.Policy.Gets(context.Background(), orgId, "", "")
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, policy := range policies.Data {
		mqlAtlassianAdminPolicy, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.policy",
			map[string]*llx.RawData{
				"id":         llx.StringData(policy.ID),
				"type":       llx.StringData(policy.Type),
				"name":       llx.StringData(policy.Attributes.Name),
				"status":     llx.StringData(policy.Attributes.Status),
				"policyType": llx.StringData(policy.Attributes.Type),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminPolicy)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) domains() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow admin access")
	}
	admin := conn.Client()
	orgId := a.Id.Data
	domains, _, err := admin.Organization.Domains(context.Background(), orgId, "")
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, domain := range domains.Data {
		mqlAtlassianAdminDomain, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.domain",
			map[string]*llx.RawData{
				"id":   llx.StringData(domain.ID),
				"name": llx.StringData(domain.Attributes.Name),
				"type": llx.StringData(domain.Type),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminDomain)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganization) id() (string, error) {
	return a.Id.Data, nil
}

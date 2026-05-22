// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/atlassian/connection/admin"
	"go.mondoo.com/mql/v13/types"
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

func (a *mqlAtlassianAdminOrganization) managedUsers() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow admin access")
	}

	admin := conn.Client()

	managedUsers, _, err := admin.Organization.Users(context.Background(), a.Id.Data, "")
	if err != nil {
		return nil, err
	}
	res := []any{}
	for _, user := range managedUsers.Data {

		type ProductAccess struct {
			Name       string
			LastActive *time.Time
		}
		var products []ProductAccess

		for i := range user.ProductAccess {

			var lastProductUse *time.Time
			if user.LastActive != "" {
				t, err := time.Parse(time.RFC3339, user.LastActive)
				if err != nil {
					lastProductUse = &t
				}
			}

			products = append(products, ProductAccess{
				Name:       user.ProductAccess[i].Name,
				LastActive: lastProductUse,
			})
		}

		var lastActive *time.Time
		if user.LastActive != "" {
			t, err := time.Parse(time.RFC3339, user.LastActive)
			if err != nil {
				lastActive = &t
			}
		}

		productArray, err := convert.JsonToDictSlice(products)
		if err != nil {
			return nil, err
		}

		mqlAtlassianAdminManagedUser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.managedUser",
			map[string]*llx.RawData{
				"id":             llx.StringData(user.AccountID),
				"name":           llx.StringData(user.Name),
				"type":           llx.StringData(user.AccountType),
				"status":         llx.StringData(user.AccountStatus),
				"email":          llx.StringData(user.Email),
				"lastActive":     llx.TimeDataPtr(lastActive),
				"productAccess":  llx.ArrayData(productArray, types.Dict),
				"organizationId": llx.StringData(a.Id.Data),
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

func (a *mqlAtlassianAdminOrganization) policies() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAtlassianAdminOrganization) domains() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow admin access")
	}
	admin := conn.Client()
	orgId := a.Id.Data
	domains, resp, err := admin.Organization.Domains(context.Background(), orgId, "")
	if err != nil && resp.StatusCode != 404 {
		a.Domains.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	} else if resp.StatusCode == 404 {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	res := []any{}
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

func (a *mqlAtlassianAdminOrganizationDomain) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganizationPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganization) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganizationManagedUser) apiTokens() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*admin.AdminConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow admin access")
	}
	client := conn.Client()

	tokens, resp, err := client.User.Token.Gets(context.Background(), a.Id.Data)
	if err != nil {
		// Some accounts (apps, customers) don't expose token APIs; surface as empty rather than fail the whole query.
		if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
			return []any{}, nil
		}
		return nil, err
	}

	res := make([]any, 0, len(tokens))
	for _, t := range tokens {
		if t == nil {
			continue
		}
		var createdAt *time.Time
		if !t.CreatedAt.IsZero() {
			createdAt = &t.CreatedAt
		}
		var lastAccess *time.Time
		if !t.LastAccess.IsZero() {
			lastAccess = &t.LastAccess
		}
		mqlToken, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.managedUser.apiToken",
			map[string]*llx.RawData{
				"id":         llx.StringData(t.ID),
				"label":      llx.StringData(t.Label),
				"createdAt":  llx.TimeDataPtr(createdAt),
				"lastAccess": llx.TimeDataPtr(lastAccess),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlToken)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationManagedUserApiToken) id() (string, error) {
	return "atlassian.admin.organization.managedUser.apiToken/" + a.Id.Data, nil
}

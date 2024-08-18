// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"github.com/microsoftgraph/msgraph-sdk-go/directory"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
)

func (m *mqlMicrosoftTenant) id() (string, error) {
	return m.Id.Data, nil
}

// Deprecated: use `microsoft.tenant` instead
func (a *mqlMicrosoft) organizations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
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
		mqlResource, err := newMicrosoftTenant(a.MqlRuntime, org)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

var tenantFields = []string{
	"id",
	"assignedPlans",
	"createdDateTime",
	"displayName",
	"verifiedDomains",
	"onPremisesSyncEnabled",
	"tenantType",
	"provisionedPlans",
}

func initMicrosoftTenant(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Organization().ByOrganizationId(conn.TenantId()).Get(ctx, &organization.OrganizationItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &organization.OrganizationItemRequestBuilderGetQueryParameters{
			Select: tenantFields,
		},
	})
	if err != nil {
		return nil, nil, transformError(err)
	}

	tenant, err := newMicrosoftTenant(runtime, resp)
	if err != nil {
		return nil, nil, err
	}
	return nil, tenant, nil
}

func newMicrosoftTenant(runtime *plugin.Runtime, org models.Organizationable) (*mqlMicrosoftTenant, error) {
	assignedPlans, err := convert.JsonToDictSlice(newAssignedPlans(org.GetAssignedPlans()))
	if err != nil {
		return nil, err
	}
	verifiedDomains, err := convert.JsonToDictSlice(newVerifiedDomains(org.GetVerifiedDomains()))
	if err != nil {
		return nil, err
	}

	provisionedPlans, err := convert.JsonToDictSlice(newProvisionedPlans(org.GetProvisionedPlans()))
	if err != nil {
		return nil, err
	}

	mqlResource, err := CreateResource(runtime, "microsoft.tenant",
		map[string]*llx.RawData{
			"id":                    llx.StringDataPtr(org.GetId()),
			"assignedPlans":         llx.DictData(assignedPlans),
			"createdDateTime":       llx.TimeDataPtr(org.GetCreatedDateTime()), // deprecated
			"name":                  llx.StringDataPtr(org.GetDisplayName()),
			"displayName":           llx.StringDataPtr(org.GetDisplayName()), // deprecated
			"verifiedDomains":       llx.DictData(verifiedDomains),
			"onPremisesSyncEnabled": llx.BoolDataPtr(org.GetOnPremisesSyncEnabled()),
			"createdAt":             llx.TimeDataPtr(org.GetCreatedDateTime()),
			"type":                  llx.StringDataPtr(org.GetTenantType()),
			"provisionedPlans":      llx.DictData(provisionedPlans),
		})
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlMicrosoftTenant), nil
}

// https://learn.microsoft.com/en-us/entra/identity/users/licensing-service-plan-reference
func (a *mqlMicrosoftTenant) subscriptions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	resp, err := graphClient.Directory().Subscriptions().Get(context.Background(), &directory.SubscriptionsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	for _, sub := range resp.GetValue() {
		res = append(res, newCompanySubscription(sub))
	}

	return convert.JsonToDictSlice(res)
}

func (a *mqlMicrosoft) tenantDomainName() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	resp, err := graphClient.Organization().Get(ctx, &organization.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return "", transformError(err)
	}
	tenantDomainName := ""

	for _, org := range resp.GetValue() {
		org.GetId()
		org.GetTenantType()
		org.GetDisplayName()
		org.GetProvisionedPlans()
		for _, d := range org.GetVerifiedDomains() {
			if *d.GetIsInitial() {
				tenantDomainName = *d.GetName()
			}
		}
	}

	return tenantDomainName, nil
}

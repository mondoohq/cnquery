// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/domains"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (m *mqlMicrosoftDomain) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftDomaindnsrecord) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoft) domains() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Domains().Get(ctx, &domains.DomainsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	domains := resp.GetValue()
	for _, domain := range domains {
		supportedServices := []interface{}{}
		for _, service := range domain.GetSupportedServices() {
			supportedServices = append(supportedServices, service)
		}
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.domain",
			map[string]*llx.RawData{
				"id":                               llx.StringDataPtr(domain.GetId()),
				"authenticationType":               llx.StringDataPtr(domain.GetAuthenticationType()),
				"availabilityStatus":               llx.StringDataPtr(domain.GetAvailabilityStatus()),
				"isAdminManaged":                   llx.BoolDataPtr(domain.GetIsAdminManaged()),
				"isDefault":                        llx.BoolDataPtr(domain.GetIsDefault()),
				"isInitial":                        llx.BoolDataPtr(domain.GetIsInitial()),
				"isRoot":                           llx.BoolDataPtr(domain.GetIsRoot()),
				"isVerified":                       llx.BoolDataPtr(domain.GetIsVerified()),
				"passwordNotificationWindowInDays": llx.IntDataDefault(domain.GetPasswordNotificationWindowInDays(), 0),
				"passwordValidityPeriodInDays":     llx.IntDataDefault(domain.GetPasswordValidityPeriodInDays(), 0),
				"supportedServices":                llx.ArrayData(supportedServices, types.String),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func (a *mqlMicrosoftDomain) serviceConfigurationRecords() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	id := a.Id.Data
	ctx := context.Background()
	resp, err := graphClient.Domains().ByDomainId(id).ServiceConfigurationRecords().Get(ctx, &domains.ItemServiceConfigurationRecordsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	records := resp.GetValue()
	for _, record := range records {
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.domaindnsrecord",
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(record.GetId()),
				"isOptional":       llx.BoolDataPtr(record.GetIsOptional()),
				"label":            llx.StringDataPtr(record.GetLabel()),
				"recordType":       llx.StringDataPtr(record.GetRecordType()),
				"supportedService": llx.StringDataPtr(record.GetSupportedService()),
				"ttl":              llx.IntDataDefault(record.GetTtl(), 0),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

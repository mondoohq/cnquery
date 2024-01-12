// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/domains"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (m *mqlMicrosoftDomain) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftDomaindnsrecord) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoft) domains() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := graphClient(conn)
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
				"id":                               llx.StringData(convert.ToString(domain.GetId())),
				"authenticationType":               llx.StringData(convert.ToString(domain.GetAuthenticationType())),
				"availabilityStatus":               llx.StringData(convert.ToString(domain.GetAvailabilityStatus())),
				"isAdminManaged":                   llx.BoolData(convert.ToBool(domain.GetIsAdminManaged())),
				"isDefault":                        llx.BoolData(convert.ToBool(domain.GetIsDefault())),
				"isInitial":                        llx.BoolData(convert.ToBool(domain.GetIsInitial())),
				"isRoot":                           llx.BoolData(convert.ToBool(domain.GetIsRoot())),
				"isVerified":                       llx.BoolData(convert.ToBool(domain.GetIsVerified())),
				"passwordNotificationWindowInDays": llx.IntData(convert.ToInt64From32(domain.GetPasswordNotificationWindowInDays())),
				"passwordValidityPeriodInDays":     llx.IntData(convert.ToInt64From32(domain.GetPasswordValidityPeriodInDays())),
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
	graphClient, err := graphClient(conn)
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
		properties := getDomainsDnsRecordProperties(record)
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.domaindnsrecord",
			map[string]*llx.RawData{
				"id":               llx.StringData(convert.ToString(record.GetId())),
				"isOptional":       llx.BoolData(convert.ToBool(record.GetIsOptional())),
				"label":            llx.StringData(convert.ToString(record.GetLabel())),
				"recordType":       llx.StringData(convert.ToString(record.GetRecordType())),
				"supportedService": llx.StringData(convert.ToString(record.GetSupportedService())),
				"ttl":              llx.IntData(convert.ToInt64From32(record.GetTtl())),
				"properties":       llx.DictData(properties),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}

	return res, nil
}

func getDomainsDnsRecordProperties(record models.DomainDnsRecordable) map[string]interface{} {
	props := map[string]interface{}{}
	if record.GetOdataType() != nil {
		props["@odata.type"] = *record.GetOdataType()
	}
	txtRecord, ok := record.(*models.DomainDnsTxtRecord)
	if ok {
		if txtRecord.GetText() != nil {
			props["text"] = *txtRecord.GetText()
		}
	}
	mxRecord, ok := record.(*models.DomainDnsMxRecord)
	if ok {
		if mxRecord.GetMailExchange() != nil {
			props["mailExchange"] = *mxRecord.GetMailExchange()
		}
		if mxRecord.GetPreference() != nil {
			props["preference"] = *mxRecord.GetPreference()
		}
	}
	cNameRecord, ok := record.(*models.DomainDnsCnameRecord)
	if ok {
		if cNameRecord.GetCanonicalName() != nil {
			props["canonicalName"] = *cNameRecord.GetCanonicalName()
		}
	}
	srvRecord, ok := record.(*models.DomainDnsSrvRecord)
	if ok {
		if srvRecord.GetNameTarget() != nil {
			props["nameTarget"] = *srvRecord.GetNameTarget()
		}
		if srvRecord.GetPort() != nil {
			props["port"] = *srvRecord.GetPort()
		}
		if srvRecord.GetPriority() != nil {
			props["priority"] = *srvRecord.GetPriority()
		}
		if srvRecord.GetProtocol() != nil {
			props["protocol"] = *srvRecord.GetProtocol()
		}
		if srvRecord.GetService() != nil {
			props["service"] = *srvRecord.GetService()
		}
		if srvRecord.GetWeight() != nil {
			props["weight"] = *srvRecord.GetWeight()
		}
	}
	return props
}

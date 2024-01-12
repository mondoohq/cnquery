// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/okta/connection"
)

func (o *mqlOkta) domains() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	domainSlice, _, err := client.Domain.ListDomains(
		ctx,
	)
	if err != nil {
		return nil, err
	}

	if len(domainSlice.Domains) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	for i := range domainSlice.Domains {
		entry := domainSlice.Domains[i]
		r, err := newMqlOktaDomain(o.MqlRuntime, entry)
		if err != nil {
			return nil, err
		}
		list = append(list, r)

	}

	return list, nil
}

func newMqlOktaDomain(runtime *plugin.Runtime, entry *okta.Domain) (interface{}, error) {
	publicCertificate, err := convert.JsonToDict(entry.PublicCertificate)
	if err != nil {
		return nil, err
	}

	dnsRecords, err := convert.JsonToDictSlice(entry.DnsRecords)

	return runtime.CreateResource(runtime, "okta.domain", map[string]*llx.RawData{
		"id":                llx.StringData(entry.Id),
		"domain":            llx.StringData(entry.Domain),
		"validationStatus":  llx.StringData(entry.ValidationStatus),
		"publicCertificate": llx.DictData(publicCertificate),
		"dnsRecords":        llx.DictData(dnsRecords),
	})
}

func (o *mqlOktaDomain) id() (string, error) {
	return "okta.domain/" + o.Id.Data, o.Id.Error
}

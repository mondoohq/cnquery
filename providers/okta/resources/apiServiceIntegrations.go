// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOkta) apiServiceIntegrations() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.ApiServiceIntegrationsAPI.ListApiServiceIntegrationInstances(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.APIServiceIntegrationInstance) error {
		for i := range datalist {
			r, err := newMqlOktaApiServiceIntegration(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.APIServiceIntegrationInstance
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaApiServiceIntegration(runtime *plugin.Runtime, entry *okta.APIServiceIntegrationInstance) (any, error) {
	// The SDK types createdAt as a plain string, but Okta returns an RFC3339
	// timestamp; parse it so the field matches the time type used elsewhere.
	var created *time.Time
	if entry.CreatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *entry.CreatedAt); err == nil {
			created = &t
		}
	}

	return CreateResource(runtime, "okta.apiServiceIntegration", map[string]*llx.RawData{
		"id":             llx.StringData(oktaStr(entry.Id)),
		"name":           llx.StringData(oktaStr(entry.Name)),
		"type":           llx.StringData(oktaStr(entry.Type)),
		"grantedScopes":  llx.ArrayData(convert.SliceAnyToInterface(entry.GrantedScopes), types.String),
		"configGuideUrl": llx.StringData(oktaStr(entry.ConfigGuideUrl)),
		"createdBy":      llx.StringData(oktaStr(entry.CreatedBy)),
		"createdAt":      llx.TimeDataPtr(created),
	})
}

func (o *mqlOktaApiServiceIntegration) id() (string, error) {
	return "okta.apiServiceIntegration/" + o.Id.Data, o.Id.Error
}

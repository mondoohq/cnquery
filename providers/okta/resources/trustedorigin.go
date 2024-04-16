// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/okta/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (o *mqlOkta) trustedOrigins() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.TrustedOrigin.ListOrigins(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.TrustedOrigin) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaTrustedOrigin(o.MqlRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(slice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var slice []*okta.TrustedOrigin
		resp, err = resp.Next(ctx, &slice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(slice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaTrustedOrigin(runtime *plugin.Runtime, entry *okta.TrustedOrigin) (interface{}, error) {
	scopes, err := convert.JsonToDictSlice(entry.Scopes)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.trustedOrigin", map[string]*llx.RawData{
		"id":            llx.StringData(entry.Id),
		"name":          llx.StringData(entry.Name),
		"origin":        llx.StringData(entry.Origin),
		"created":       llx.TimeDataPtr(entry.Created),
		"createdBy":     llx.StringData(entry.CreatedBy),
		"lastUpdated":   llx.TimeDataPtr(entry.LastUpdated),
		"lastUpdatedBy": llx.StringData(entry.LastUpdatedBy),
		"scopes":        llx.ArrayData(scopes, types.Dict),
		"status":        llx.StringData(entry.Status),
	})
}

func (o *mqlOktaTrustedOrigin) id() (string, error) {
	return "okta.trustedOriogin/" + o.Id.Data, o.Id.Error
}

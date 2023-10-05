// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/okta/connection"
)

func (o *mqlOkta) groups() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.Group.ListGroups(
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
	appendEntry := func(datalist []*okta.Group) error {
		for i := range datalist {
			entry := datalist[i]
			r, err := newMqlOktaGroup(o.MqlRuntime, entry)
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
		var slice []*okta.Group
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

func newMqlOktaGroup(runtime *plugin.Runtime, entry *okta.Group) (interface{}, error) {
	profile, err := convert.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.group", map[string]*llx.RawData{
		"id":                    llx.StringData(entry.Id),
		"type":                  llx.StringData(entry.Type),
		"created":               llx.TimeDataPtr(entry.Created),
		"lastMembershipUpdated": llx.TimeDataPtr(entry.LastMembershipUpdated),
		"lastUpdated":           llx.TimeDataPtr(entry.LastUpdated),
		"profile":               llx.DictData(profile),
	})
}

func (o *mqlOktaGroup) id() (string, error) {
	return "okta.group/" + o.Id.Data, o.Id.Error
}

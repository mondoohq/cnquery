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

func (o *mqlOkta) applications() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	appSetSlice, resp, err := client.Application.ListApplications(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(appSetSlice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []okta.App) error {
		for i := range datalist {
			entry := datalist[i]
			if entry.IsApplicationInstance() {
				app := entry.(*okta.Application)
				r, err := newMqlOktaApplication(o.MqlRuntime, app)
				if err != nil {
					return err
				}
				list = append(list, r)
			}
		}
		return nil
	}

	err = appendEntry(appSetSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var userSetSlice []okta.App
		resp, err = resp.Next(ctx, &userSetSlice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(userSetSlice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaApplication(runtime *plugin.Runtime, entry *okta.Application) (interface{}, error) {
	credentials, err := convert.JsonToDict(entry.Credentials)
	if err != nil {
		return nil, err
	}

	licensing, err := convert.JsonToDict(entry.Licensing)
	if err != nil {
		return nil, err
	}

	profile, err := convert.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	settings, err := convert.JsonToDict(entry.Settings)
	if err != nil {
		return nil, err
	}

	visibility, err := convert.JsonToDict(entry.Visibility)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.application", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"label":       llx.StringData(entry.Label),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
		"credentials": llx.DictData(credentials),
		"features":    llx.ArrayData(convert.SliceAnyToInterface(entry.Features), types.String),
		"licensing":   llx.DictData(licensing),
		"profile":     llx.DictData(profile),
		"settings":    llx.DictData(settings),
		"signOnMode":  llx.StringData(entry.SignOnMode),
		"status":      llx.StringData(entry.Status),
		"visibility":  llx.DictData(visibility),
	})
}

func (o *mqlOktaApplication) id() (string, error) {
	return "okta.application/" + o.Id.Data, o.Id.Error
}

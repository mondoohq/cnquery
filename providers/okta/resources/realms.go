// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

func (o *mqlOkta) realms() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.RealmAPI.ListRealms(ctx).Limit(queryLimit).Execute()
	if err != nil {
		// Realms are a licensed feature; orgs without it have no endpoint.
		if isOktaFeatureUnavailable(resp, err) {
			return nil, nil
		}
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.Realm) error {
		for i := range datalist {
			r, err := newMqlOktaRealm(o.MqlRuntime, &datalist[i])
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
		var page []okta.Realm
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

func newMqlOktaRealm(runtime *plugin.Runtime, entry *okta.Realm) (any, error) {
	var name, realmType string
	if profile := entry.Profile; profile != nil {
		name = profile.Name
		realmType = oktaStr(profile.RealmType)
	}

	return CreateResource(runtime, "okta.realm", map[string]*llx.RawData{
		"id":          llx.StringData(oktaStr(entry.Id)),
		"name":        llx.StringData(name),
		"realmType":   llx.StringData(realmType),
		"isDefault":   llx.BoolDataPtr(entry.IsDefault),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaRealm) id() (string, error) {
	return "okta.realm/" + o.Id.Data, o.Id.Error
}

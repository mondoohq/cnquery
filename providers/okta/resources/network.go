// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/okta/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (o *mqlOkta) networks() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	networkSlice, resp, err := client.NetworkZone.ListNetworkZones(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(networkSlice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.NetworkZone) error {
		for i := range datalist {
			entry := datalist[i]

			r, err := newMqlOktaNetworkZone(o.MqlRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(networkSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var networkSlice []*okta.NetworkZone
		resp, err = resp.Next(ctx, &networkSlice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(networkSlice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaNetworkZone(runtime *plugin.Runtime, entry *okta.NetworkZone) (interface{}, error) {
	proxies, err := convert.JsonToDictSlice(entry.Proxies)
	if err != nil {
		return nil, err
	}

	locations, err := convert.JsonToDictSlice(entry.Locations)
	if err != nil {
		return nil, err
	}

	gateways, err := convert.JsonToDictSlice(entry.Gateways)
	if err != nil {
		return nil, err
	}

	system := false
	if entry.System != nil {
		system = *entry.System
	}

	return CreateResource(runtime, "okta.network", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"type":        llx.StringData(entry.Type),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
		"status":      llx.StringData(entry.Status),
		"system":      llx.BoolData(system),
		"asns":        llx.ArrayData(convert.SliceAnyToInterface(entry.Asns), types.String),
		"usage":       llx.StringData(entry.Usage),
		"proxyType":   llx.StringData(entry.ProxyType),
		"proxies":     llx.ArrayData(proxies, types.Dict),
		"locations":   llx.ArrayData(locations, types.Dict),
		"gateways":    llx.ArrayData(gateways, types.Dict),
	})
}

func (o *mqlOktaNetwork) id() (string, error) {
	return "okta.network/" + o.Id.Data, o.Id.Error
}

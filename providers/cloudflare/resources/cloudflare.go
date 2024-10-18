// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (r *mqlCloudflare) id() (string, error) {
	return "cloudflare", nil
}

func (c *mqlCloudflare) zones() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	zones, err := conn.Cf.ListZones(context.Background())
	if err != nil {
		return nil, err
	}

	var res []any
	for i := range zones {
		zone := zones[i]

		acc, err := NewResource(c.MqlRuntime, "cloudflare.zone.account", map[string]*llx.RawData{
			"id":   llx.StringData(zone.Account.ID),
			"name": llx.StringData(zone.Account.Name),
			"type": llx.StringData(zone.Account.Type),
		})
		if err != nil {
			return nil, err
		}

		r, err := NewResource(c.MqlRuntime, "cloudflare.zone", map[string]*llx.RawData{
			"id":   llx.StringData(zone.ID),
			"name": llx.StringData(zone.Name),

			"name_servers":          llx.ArrayData(convert.SliceAnyToInterface(zone.NameServers), types.String),
			"original_name_servers": llx.ArrayData(convert.SliceAnyToInterface(zone.OriginalNS), types.String),

			"status": llx.StringData(zone.Status),
			"paused": llx.BoolData(zone.Paused),
			"type":   llx.StringData(zone.Type),

			"account": llx.ResourceData(acc, acc.MqlName()),

			"created_on":  llx.TimeData(zone.CreatedOn),
			"modified_on": llx.TimeData(zone.ModifiedOn),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (c *mqlCloudflare) accounts() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	cursor := cloudflare.ResultInfo{}

	for {
		_accounts, info, err := conn.Cf.Accounts(context.Background(), cloudflare.AccountsListParams{
			PaginationOptions: cloudflare.PaginationOptions{
				Page:    cursor.Page,
				PerPage: cursor.PerPage,
			},
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range _accounts {
			acc := _accounts[i]

			settings, err := NewResource(c.MqlRuntime, "cloudflare.account.settings", map[string]*llx.RawData{
				"enforce_two_factor": llx.BoolData(acc.Settings.EnforceTwoFactor),
			})
			if err != nil {
				return nil, err
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.account", map[string]*llx.RawData{
				"id":         llx.StringData(acc.ID),
				"name":       llx.StringData(acc.Name),
				"settings":   llx.ResourceData(settings, settings.MqlName()),
				"created_on": llx.TimeData(acc.CreatedOn),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !cursor.HasMorePages() {
			break
		}
	}

	return result, nil
}

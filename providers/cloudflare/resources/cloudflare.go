// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

const defaultCloudflarePageSize = 50

// paginateRaw walks pagination on a Cloudflare GET endpoint via the SDK's Raw
// helper, which reuses the same HTTP client, retries, and authentication as
// the typed SDK methods. Stops when the response omits result_info or
// HasMorePages reports no further pages. handle is invoked once per page with
// the raw `result` array bytes.
//
// This exists because many cloudflare-go v0 List* methods (devices, posture
// rules, notifications, teams rules/lists/locations, DLP profiles, workers)
// take no pagination parameters and silently return only the first page.
func paginateRaw(ctx context.Context, cf *cloudflare.API, endpoint string, handle func(raw json.RawMessage) error) error {
	page := 1
	for {
		sep := "?"
		if strings.Contains(endpoint, "?") {
			sep = "&"
		}
		uri := fmt.Sprintf("%s%spage=%d&per_page=%d", endpoint, sep, page, defaultCloudflarePageSize)
		raw, err := cf.Raw(ctx, http.MethodGet, uri, nil, nil)
		if err != nil {
			return err
		}
		if err := handle(raw.Result); err != nil {
			return err
		}
		if raw.ResultInfo == nil || !raw.ResultInfo.HasMorePages() {
			return nil
		}
		page = raw.ResultInfo.Page + 1
	}
}

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

		owner, err := NewResource(c.MqlRuntime, "cloudflare.zone.owner", map[string]*llx.RawData{
			"id":        llx.StringData(zone.Owner.ID),
			"email":     llx.StringData(zone.Owner.Email),
			"name":      llx.StringData(zone.Owner.Name),
			"ownerType": llx.StringData(zone.Owner.OwnerType),
		})
		if err != nil {
			return nil, err
		}

		plan, err := NewResource(c.MqlRuntime, "cloudflare.zone.plan", map[string]*llx.RawData{
			"id":           llx.StringData(zone.Plan.ID),
			"name":         llx.StringData(zone.Plan.Name),
			"price":        llx.IntData(zone.Plan.Price),
			"currency":     llx.StringData(zone.Plan.Currency),
			"frequency":    llx.StringData(zone.Plan.Frequency),
			"isSubscribed": llx.BoolData(zone.Plan.IsSubscribed),
		})
		if err != nil {
			return nil, err
		}

		r, err := NewResource(c.MqlRuntime, "cloudflare.zone", map[string]*llx.RawData{
			"id":   llx.StringData(zone.ID),
			"name": llx.StringData(zone.Name),

			"nameServers":         llx.ArrayData(convert.SliceAnyToInterface(zone.NameServers), types.String),
			"originalNameServers": llx.ArrayData(convert.SliceAnyToInterface(zone.OriginalNS), types.String),

			"status": llx.StringData(zone.Status),
			"paused": llx.BoolData(zone.Paused),
			"type":   llx.StringData(zone.Type),

			"account": llx.ResourceData(acc, acc.MqlName()),
			"owner":   llx.ResourceData(owner, owner.MqlName()),
			"plan":    llx.ResourceData(plan, plan.MqlName()),

			"createdOn":  llx.TimeData(zone.CreatedOn),
			"modifiedOn": llx.TimeData(zone.ModifiedOn),
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
	params := cloudflare.AccountsListParams{}

	for {
		_accounts, info, err := conn.Cf.Accounts(context.Background(), params)
		if err != nil {
			return nil, err
		}

		for i := range _accounts {
			acc := _accounts[i]

			settingsArgs := map[string]*llx.RawData{
				"__id": llx.StringData("cloudflare.account.settings@" + acc.ID),
			}
			if acc.Settings != nil {
				settingsArgs["enforceTwoFactor"] = llx.BoolData(acc.Settings.EnforceTwoFactor)
			} else {
				settingsArgs["enforceTwoFactor"] = llx.BoolData(false)
			}
			settings, err := NewResource(c.MqlRuntime, "cloudflare.account.settings", settingsArgs)
			if err != nil {
				return nil, err
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.account", map[string]*llx.RawData{
				"id":        llx.StringData(acc.ID),
				"name":      llx.StringData(acc.Name),
				"type":      llx.StringData(acc.Type),
				"settings":  llx.ResourceData(settings, settings.MqlName()),
				"createdOn": llx.TimeData(acc.CreatedOn),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !info.HasMorePages() {
			break
		}
		params.PaginationOptions.Page = info.Page + 1
	}

	return result, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareNotificationPolicy) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return "cloudflare/notificationPolicy/" + c.Id.Data, nil
}

func (c *mqlCloudflareAccount) notificationPolicies() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	resp, err := conn.Cf.ListNotificationPolicies(context.TODO(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	// The cloudflare-go v0.117.0 SDK does not expose pagination on
	// ListNotificationPolicies, even though the underlying API does. If the
	// response is truncated, warn so the operator knows the result set may
	// be incomplete.
	if resp.ResultInfo.HasMorePages() {
		log.Warn().
			Int("returned", len(resp.Result)).
			Int("total", resp.ResultInfo.Total).
			Str("account", c.Id.Data).
			Msg("cloudflare> notification policies truncated; SDK does not support pagination on this endpoint")
	}

	results := []any{}
	for i := range resp.Result {
		p := resp.Result[i]

		mechanisms := map[string]any{}
		for name, integrations := range p.Mechanisms {
			list := make([]any, 0, len(integrations))
			for _, m := range integrations {
				list = append(list, map[string]any{
					"id":   m.ID,
					"name": m.Name,
				})
			}
			mechanisms[name] = list
		}

		filters := map[string]any{}
		for k, v := range p.Filters {
			vv := make([]any, 0, len(v))
			for _, s := range v {
				vv = append(vv, s)
			}
			filters[k] = vv
		}

		res, err := NewResource(c.MqlRuntime, "cloudflare.notificationPolicy", map[string]*llx.RawData{
			"id":          llx.StringData(p.ID),
			"name":        llx.StringData(p.Name),
			"description": llx.StringData(p.Description),
			"enabled":     llx.BoolData(p.Enabled),
			"alertType":   llx.StringData(p.AlertType),
			"mechanisms":  llx.DictData(mechanisms),
			"conditions":  llx.DictData(anyMap(p.Conditions)),
			"filters":     llx.DictData(filters),
			"created":     llx.TimeData(p.Created),
			"modified":    llx.TimeData(p.Modified),
		})
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

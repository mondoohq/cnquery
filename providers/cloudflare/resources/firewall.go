// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

func (c *mqlCloudflareZoneFirewallRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) firewallRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}
	var result []any
	for {
		records, info, err := conn.Cf.FirewallRules(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.Id.Data,
			Level:      cloudflare.ZoneRouteLevel,
		}, cloudflare.FirewallRuleListParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.zone.firewallRule", map[string]*llx.RawData{
				"id":               llx.StringData(rec.ID),
				"description":      llx.StringData(rec.Description),
				"action":           llx.StringData(rec.Action),
				"ref":              llx.StringData(rec.Ref),
				"paused":           llx.BoolData(rec.Paused),
				"filterExpression": llx.StringData(rec.Filter.Expression),
				"products":         llx.ArrayData(convert.SliceAnyToInterface(rec.Products), types.String),
				"createdAt":        llx.TimeData(rec.CreatedOn),
				"updatedAt":        llx.TimeData(rec.ModifiedOn),
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

func (c *mqlCloudflareZoneRuleset) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) rulesets() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, err := conn.Cf.ListRulesets(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.Id.Data,
		Level:      cloudflare.ZoneRouteLevel,
	}, cloudflare.ListRulesetsParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.ruleset", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"name":        llx.StringData(rec.Name),
			"description": llx.StringData(rec.Description),
			"kind":        llx.StringData(rec.Kind),
			"phase":       llx.StringData(rec.Phase),
			"version":     llx.StringDataPtr(rec.Version),
			"updatedAt":   llx.TimeDataPtr(rec.LastUpdated),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

func (c *mqlCloudflareZonePageRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) pageRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, err := conn.Cf.ListPageRules(context.TODO(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.pageRule", map[string]*llx.RawData{
			"id":        llx.StringData(rec.ID),
			"status":    llx.StringData(rec.Status),
			"priority":  llx.IntData(int64(rec.Priority)),
			"createdAt": llx.TimeData(rec.CreatedOn),
			"updatedAt": llx.TimeData(rec.ModifiedOn),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

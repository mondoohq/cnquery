// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/page_rules"
	"github.com/cloudflare/cloudflare-go/v6/rulesets"
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

// firewallRule is the response shape of the legacy zone firewall-rules endpoint
// (`/zones/{id}/firewall/rules`). cloudflare-go v6 marks its typed
// firewall.RuleService as deprecated and its typed response drops the
// created/modified timestamps we expose, so we read the endpoint via the
// client's generic Get and decode the full payload ourselves to preserve the
// MQL schema.
type firewallRule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Ref         string `json:"ref"`
	Paused      bool   `json:"paused"`
	Filter      struct {
		Expression string `json:"expression"`
	} `json:"filter"`
	Products   []string  `json:"products"`
	CreatedOn  time.Time `json:"created_on"`
	ModifiedOn time.Time `json:"modified_on"`
}

func (c *mqlCloudflareZone) firewallRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	page := 1
	for {
		var env struct {
			Result     []firewallRule `json:"result"`
			ResultInfo struct {
				Page       int `json:"page"`
				TotalPages int `json:"total_pages"`
			} `json:"result_info"`
		}
		uri := fmt.Sprintf("zones/%s/firewall/rules?page=%d&per_page=100", c.Id.Data, page)
		if err := conn.Cf.Get(context.TODO(), uri, nil, &env); err != nil {
			return nil, err
		}

		for i := range env.Result {
			rec := env.Result[i]

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

		if env.ResultInfo.TotalPages == 0 || env.ResultInfo.Page >= env.ResultInfo.TotalPages {
			break
		}
		page++
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

	var result []any
	iter := conn.Cf.Rulesets.ListAutoPaging(context.TODO(), rulesets.RulesetListParams{
		ZoneID: cloudflare.F(c.Id.Data),
	})
	for iter.Next() {
		rec := iter.Current()

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.ruleset", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"name":        llx.StringData(rec.Name),
			"description": llx.StringData(rec.Description),
			"kind":        llx.StringData(string(rec.Kind)),
			"phase":       llx.StringData(string(rec.Phase)),
			"version":     llx.StringData(rec.Version),
			"updatedAt":   timeOrNil(rec.LastUpdated),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		return nil, err
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

	records, err := conn.Cf.PageRules.List(context.TODO(), page_rules.PageRuleListParams{
		ZoneID: cloudflare.F(c.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range *records {
		rec := (*records)[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.pageRule", map[string]*llx.RawData{
			"id":        llx.StringData(rec.ID),
			"status":    llx.StringData(string(rec.Status)),
			"priority":  llx.IntData(rec.Priority),
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

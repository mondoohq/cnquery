// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

func (c *mqlCloudflareZoneRateLimitRule) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) rateLimitRules() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	limits, err := conn.Cf.ListAllRateLimits(context.TODO(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range limits {
		l := limits[i]

		action := map[string]any{
			"mode":    l.Action.Mode,
			"timeout": int64(l.Action.Timeout),
		}
		if l.Action.Response != nil {
			action["response"] = map[string]any{
				"contentType": l.Action.Response.ContentType,
				"body":        l.Action.Response.Body,
			}
		}

		statuses := make([]any, 0, len(l.Match.Response.Statuses))
		for _, s := range l.Match.Response.Statuses {
			statuses = append(statuses, int64(s))
		}

		res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.rateLimitRule", map[string]*llx.RawData{
			"__id":             llx.StringData("cloudflare.zone.rateLimitRule@" + c.Id.Data + "/" + l.ID),
			"id":               llx.StringData(l.ID),
			"description":      llx.StringData(l.Description),
			"disabled":         llx.BoolData(l.Disabled),
			"threshold":        llx.IntData(int64(l.Threshold)),
			"period":           llx.IntData(int64(l.Period)),
			"action":           llx.DictData(action),
			"urlPattern":       llx.StringData(l.Match.Request.URLPattern),
			"methods":          llx.ArrayData(convert.SliceAnyToInterface(l.Match.Request.Methods), types.String),
			"schemes":          llx.ArrayData(convert.SliceAnyToInterface(l.Match.Request.Schemes), types.String),
			"responseStatuses": llx.ArrayData(statuses, types.Int),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
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

	results := []any{}
	err := paginateRaw(context.TODO(), conn.Cf, fmt.Sprintf("/accounts/%s/alerting/v3/policies", c.Id.Data), func(raw json.RawMessage) error {
		var policies []cloudflare.NotificationPolicy
		if err := json.Unmarshal(raw, &policies); err != nil {
			return err
		}
		for i := range policies {
			p := policies[i]

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
				return err
			}
			results = append(results, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareAccountAuditLog) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return "cloudflare/account/auditLog/" + c.Id.Data, nil
}

// auditLogs returns audit log entries for the account (most recent first).
// Cloudflare's audit log retention is bounded server-side; this walks every
// page of results until the API reports no more pages.
func (c *mqlCloudflareAccount) auditLogs() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	results := []any{}
	filter := cloudflare.AuditLogFilter{
		PerPage:   100,
		Direction: "desc",
		Page:      1,
	}
	for {
		resp, err := conn.Cf.GetOrganizationAuditLogs(context.TODO(), c.Id.Data, filter)
		if err != nil {
			return nil, err
		}

		for i := range resp.Result {
			entry := resp.Result[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.account.auditLog", map[string]*llx.RawData{
				"id":           llx.StringData(entry.ID),
				"when":         llx.TimeData(entry.When),
				"actorEmail":   llx.StringData(entry.Actor.Email),
				"actorId":      llx.StringData(entry.Actor.ID),
				"actorIp":      llx.StringData(entry.Actor.IP),
				"actorType":    llx.StringData(entry.Actor.Type),
				"actionType":   llx.StringData(entry.Action.Type),
				"actionResult": llx.BoolData(entry.Action.Result),
				"resourceId":   llx.StringData(entry.Resource.ID),
				"resourceType": llx.StringData(entry.Resource.Type),
				"ownerId":      llx.StringData(entry.Owner.ID),
				"oldValue":     llx.DictData(anyMap(entry.OldValueJSON)),
				"newValue":     llx.DictData(anyMap(entry.NewValueJSON)),
			})
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}

		if !resp.ResultInfo.HasMorePages() {
			break
		}
		filter.Page = resp.ResultInfo.Page + 1
	}
	return results, nil
}

// anyMap returns an empty map[string]any when v is nil so DictData stays
// non-nil; the .lr `dict` field type tolerates nil but a stable empty map
// reads more cleanly in MQL queries.
func anyMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

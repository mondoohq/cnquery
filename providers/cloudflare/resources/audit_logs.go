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

// auditLogs returns the first page of audit log entries for the account
// (most recent first). The Cloudflare API caps a page at 100 entries; callers
// that need deeper history should filter by `when` or page through the API
// outside MQL.
func (c *mqlCloudflareAccount) auditLogs() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	resp, err := conn.Cf.GetOrganizationAuditLogs(context.TODO(), c.Id.Data, cloudflare.AuditLogFilter{
		PerPage:   100,
		Direction: "desc",
	})
	if err != nil {
		return nil, err
	}

	results := []any{}
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

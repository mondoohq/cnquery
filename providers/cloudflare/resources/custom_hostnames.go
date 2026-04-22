// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareZoneCustomHostname) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) customHostnames() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var result []any
	page := 1
	for {
		records, info, err := conn.Cf.CustomHostnames(context.TODO(), c.Id.Data, page, cloudflare.CustomHostname{})
		if err != nil {
			return nil, err
		}

		for i := range records {
			rec := records[i]

			sslStatus := ""
			sslMethod := ""
			sslType := ""
			if rec.SSL != nil {
				sslStatus = rec.SSL.Status
				sslMethod = rec.SSL.Method
				sslType = rec.SSL.Type
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.zone.customHostname", map[string]*llx.RawData{
				"id":                 llx.StringData(rec.ID),
				"hostname":           llx.StringData(rec.Hostname),
				"customOriginServer": llx.StringData(rec.CustomOriginServer),
				"customOriginSni":    llx.StringData(rec.CustomOriginSNI),
				"status":             llx.StringData(string(rec.Status)),
				"sslStatus":          llx.StringData(sslStatus),
				"sslMethod":          llx.StringData(sslMethod),
				"sslType":            llx.StringData(sslType),
				"createdAt":          llx.TimeDataPtr(rec.CreatedAt),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !info.HasMorePages() {
			break
		}
		page++
	}

	return result, nil
}

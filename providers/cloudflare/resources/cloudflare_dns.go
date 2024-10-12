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

type mqlCloudflareDnsInternal struct {
	ZoneID string
}

func (c *mqlCloudflareZone) dns() (*mqlCloudflareDns, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.dns", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.dns"),
	})
	if err != nil {
		return nil, err
	}

	dns := res.(*mqlCloudflareDns)
	dns.ZoneID = c.Id.Data

	return dns, nil
}

func (c *mqlCloudflareDnsRecord) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareDns) records() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}

	var result []any
	for {
		records, info, err := conn.Cf.ListDNSRecords(
			context.Background(),
			&cloudflare.ResourceContainer{Identifier: c.ZoneID}, cloudflare.ListDNSRecordsParams{
				ResultInfo: *cursor,
			})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]
			res, err := NewResource(c.MqlRuntime, "cloudflare.dns.record", map[string]*llx.RawData{
				"id":        llx.StringData(rec.ID),
				"name":      llx.StringData(rec.Name),
				"tags":      llx.ArrayData(convert.SliceAnyToInterface(rec.Tags), types.String),
				"proxied":   llx.BoolDataPtr(rec.Proxied),
				"proxiable": llx.BoolData(rec.Proxiable),
				"comment":   llx.StringData(rec.Comment),

				"type":    llx.StringData(rec.Type),
				"content": llx.StringData(rec.Content),
				"ttl":     llx.IntData(rec.TTL),

				"created_on":  llx.TimeData(rec.CreatedOn),
				"modified_on": llx.TimeData(rec.ModifiedOn),
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

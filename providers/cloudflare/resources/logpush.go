// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

type mqlCloudflareZoneLogpushJobInternal struct {
	zoneID string
}

func (c *mqlCloudflareZoneLogpushJob) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return fmt.Sprintf("logpush@%s@%d", c.zoneID, c.Id.Data), nil
}

func (c *mqlCloudflareZone) logpushJobs() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	records, err := conn.Cf.ListLogpushJobs(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.Id.Data,
		Level:      cloudflare.ZoneRouteLevel,
	}, cloudflare.ListLogpushJobsParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.logpushJob", map[string]*llx.RawData{
			"id":              llx.IntData(int64(rec.ID)),
			"name":            llx.StringData(rec.Name),
			"dataset":         llx.StringData(rec.Dataset),
			"logpullOptions":  llx.StringData(rec.LogpullOptions),
			"destinationConf": llx.StringData(rec.DestinationConf),
			"frequency":       llx.StringData(rec.Frequency),
			"errorMessage":    llx.StringData(rec.ErrorMessage),
			"enabled":         llx.BoolData(rec.Enabled),
			"lastComplete":    llx.TimeDataPtr(rec.LastComplete),
			"lastError":       llx.TimeDataPtr(rec.LastError),
		})
		if err != nil {
			return nil, err
		}

		mqlJob := res.(*mqlCloudflareZoneLogpushJob)
		mqlJob.zoneID = c.Id.Data

		result = append(result, res)
	}

	return result, nil
}

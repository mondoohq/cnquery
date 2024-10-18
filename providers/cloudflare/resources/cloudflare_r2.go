// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
)

func (c *mqlCloudflareR2) id() (string, error) {
	return "cloudflare.r2", nil
}

type mqlCloudflareR2Internal struct {
	AccountID string
}

func (c *mqlCloudflareZone) r2() (*mqlCloudflareR2, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.r2", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.r2"),
	})
	if err != nil {
		return nil, err
	}

	r2 := res.(*mqlCloudflareR2)
	r2.AccountID = c.GetAccount().Data.GetId().Data

	return r2, nil
}

func (c *mqlCloudflareR2) buckets() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	buckets, err := conn.Cf.ListR2Buckets(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.mqlCloudflareR2Internal.AccountID,
	}, cloudflare.ListR2BucketsParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range buckets {
		bucket := buckets[i]
		res, err := NewResource(c.MqlRuntime, "cloudflare.r2.bucket", map[string]*llx.RawData{
			"name":      llx.StringData(bucket.Name),
			"location":  llx.StringData(bucket.Location),
			"createdOn": llx.TimeData(*bucket.CreationDate),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareR2) id() (string, error) {
	return "cloudflare.r2", nil
}

type mqlCloudflareR2Internal struct {
	AccountID string
}

func (c *mqlCloudflareZone) r2() (*mqlCloudflareR2, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.r2", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.r2@" + c.GetAccount().Data.GetId().Data),
	})
	if err != nil {
		return nil, err
	}

	r2 := res.(*mqlCloudflareR2)
	r2.AccountID = c.GetAccount().Data.GetId().Data

	return r2, nil
}

type mqlCloudflareR2BucketInternal struct {
	accountID string
}

func (c *mqlCloudflareR2Bucket) id() (string, error) {
	if c.accountID == "" {
		return c.GetName().Data, nil
	}
	return c.accountID + "/" + c.GetName().Data, nil
}

// buckets enumerates R2 buckets across the account. cloudflare-go's
// ListR2Buckets returns only the first page (no cursor handling), so we walk
// the API directly via api.Raw and follow `result_info.cursor`.
func (c *mqlCloudflareR2) buckets() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	accountID := c.mqlCloudflareR2Internal.AccountID

	var (
		result  []any
		cursor  string
		perPage = 100
	)

	for {
		uri := fmt.Sprintf("/accounts/%s/r2/buckets?per_page=%d", accountID, perPage)
		if cursor != "" {
			uri += "&cursor=" + cursor
		}
		raw, err := conn.Cf.Raw(context.TODO(), http.MethodGet, uri, nil, nil)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Buckets []cloudflare.R2Bucket `json:"buckets"`
		}
		if len(raw.Result) > 0 {
			if err := json.Unmarshal(raw.Result, &payload); err != nil {
				return nil, fmt.Errorf("failed to decode r2 buckets response: %w", err)
			}
		}

		for i := range payload.Buckets {
			bucket := payload.Buckets[i]
			res, err := CreateResource(c.MqlRuntime, "cloudflare.r2.bucket", map[string]*llx.RawData{
				"__id":      llx.StringData(accountID + "/" + bucket.Name),
				"name":      llx.StringData(bucket.Name),
				"location":  llx.StringData(bucket.Location),
				"createdOn": llx.TimeDataPtr(bucket.CreationDate),
			})
			if err != nil {
				return nil, err
			}

			mqlBucket := res.(*mqlCloudflareR2Bucket)
			mqlBucket.accountID = accountID

			result = append(result, res)
		}

		if raw.ResultInfo == nil {
			break
		}
		next := raw.ResultInfo.Cursor
		if next == "" {
			next = raw.ResultInfo.Cursors.After
		}
		if next == "" || next == cursor {
			break
		}
		cursor = next
	}

	return result, nil
}

// publicAccess returns the bucket's managed-domain (r2.dev) public-access
// configuration. The cloudflare-go SDK does not yet wrap this endpoint, so we
// hit `/accounts/{id}/r2/buckets/{name}/domains/managed` via api.Raw.
func (c *mqlCloudflareR2Bucket) publicAccess() (*mqlCloudflareR2BucketPublicAccess, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	if c.accountID == "" {
		c.PublicAccess.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	uri := fmt.Sprintf("/accounts/%s/r2/buckets/%s/domains/managed", c.accountID, c.GetName().Data)
	raw, err := conn.Cf.Raw(context.TODO(), http.MethodGet, uri, nil, nil)
	if err != nil {
		var notFound *cloudflare.NotFoundError
		var authN *cloudflare.AuthenticationError
		var authZ *cloudflare.AuthorizationError
		if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
			c.PublicAccess.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	var payload struct {
		Enabled bool   `json:"enabled"`
		Domain  string `json:"domain"`
	}
	if len(raw.Result) > 0 {
		if err := json.Unmarshal(raw.Result, &payload); err != nil {
			return nil, fmt.Errorf("failed to decode r2 managed-domain response: %w", err)
		}
	}
	enabled, domain := payload.Enabled, payload.Domain

	res, err := CreateResource(c.MqlRuntime, "cloudflare.r2.bucket.publicAccess", map[string]*llx.RawData{
		"__id":    llx.StringData("cloudflare.r2.bucket.publicAccess@" + c.accountID + "/" + c.GetName().Data),
		"enabled": llx.BoolData(enabled),
		"domain":  llx.StringData(domain),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlCloudflareR2BucketPublicAccess), nil
}

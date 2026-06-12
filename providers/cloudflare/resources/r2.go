// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"
	"sync"

	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/r2"
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

	publicAccessLock        sync.Mutex
	publicAccessFetched     bool
	publicAccessAvailable   bool
	cachePublicAccessOn     bool
	cachePublicAccessDomain string
}

func (c *mqlCloudflareR2Bucket) id() (string, error) {
	if c.accountID == "" {
		return c.GetName().Data, nil
	}
	return c.accountID + "/" + c.GetName().Data, nil
}

// buckets enumerates R2 buckets across the account. The cloudflare-go v6 typed
// bucket list response doesn't surface the pagination cursor, so we call the
// endpoint directly via the client's generic Get and follow
// `result_info.cursor` to walk every page.
func (c *mqlCloudflareR2) buckets() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	accountID := c.mqlCloudflareR2Internal.AccountID

	var (
		result  []any
		cursor  string
		perPage = 100
	)

	for {
		uri := fmt.Sprintf("accounts/%s/r2/buckets?per_page=%d", accountID, perPage)
		if cursor != "" {
			uri += "&cursor=" + cursor
		}

		var env struct {
			Result struct {
				Buckets []r2.Bucket `json:"buckets"`
			} `json:"result"`
			ResultInfo struct {
				Cursor  string `json:"cursor"`
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
			} `json:"result_info"`
		}
		if err := conn.Cf.Get(context.TODO(), uri, nil, &env); err != nil {
			return nil, err
		}

		for i := range env.Result.Buckets {
			bucket := env.Result.Buckets[i]
			res, err := CreateResource(c.MqlRuntime, "cloudflare.r2.bucket", map[string]*llx.RawData{
				"__id":      llx.StringData(accountID + "/" + bucket.Name),
				"name":      llx.StringData(bucket.Name),
				"location":  llx.StringData(string(bucket.Location)),
				"createdOn": timeOrNil(parseRFC3339(bucket.CreationDate)),
			})
			if err != nil {
				return nil, err
			}

			mqlBucket := res.(*mqlCloudflareR2Bucket)
			mqlBucket.accountID = accountID

			result = append(result, res)
		}

		next := env.ResultInfo.Cursor
		if next == "" {
			next = env.ResultInfo.Cursors.After
		}
		if next == "" || next == cursor {
			break
		}
		cursor = next
	}

	return result, nil
}

// fetchPublicAccess fetches the bucket's managed-domain (r2.dev) public-access
// configuration. The `available` return is false when the bucket has no managed
// domain or the caller lacks access to read it; in that case the calling
// computed method should mark its field null.
func (c *mqlCloudflareR2Bucket) fetchPublicAccess() (available, enabled bool, domain string, err error) {
	if c.publicAccessFetched {
		return c.publicAccessAvailable, c.cachePublicAccessOn, c.cachePublicAccessDomain, nil
	}
	c.publicAccessLock.Lock()
	defer c.publicAccessLock.Unlock()
	if c.publicAccessFetched {
		return c.publicAccessAvailable, c.cachePublicAccessOn, c.cachePublicAccessDomain, nil
	}

	if c.accountID == "" {
		c.publicAccessFetched = true
		return false, false, "", nil
	}

	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	resp, rerr := conn.Cf.R2.Buckets.Domains.Managed.List(context.TODO(), c.GetName().Data, r2.BucketDomainManagedListParams{
		AccountID: cloudflare.F(c.accountID),
	})
	if rerr != nil {
		if isUnavailable(rerr) {
			c.publicAccessFetched = true
			return false, false, "", nil
		}
		return false, false, "", rerr
	}

	c.publicAccessAvailable = true
	c.cachePublicAccessOn = resp.Enabled
	c.cachePublicAccessDomain = resp.Domain
	c.publicAccessFetched = true
	return c.publicAccessAvailable, c.cachePublicAccessOn, c.cachePublicAccessDomain, nil
}

func (c *mqlCloudflareR2Bucket) publicAccessEnabled() (bool, error) {
	available, enabled, _, err := c.fetchPublicAccess()
	if err != nil {
		return false, err
	}
	if !available {
		c.PublicAccessEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	return enabled, nil
}

func (c *mqlCloudflareR2Bucket) publicAccessDomain() (string, error) {
	available, _, domain, err := c.fetchPublicAccess()
	if err != nil {
		return "", err
	}
	if !available {
		c.PublicAccessDomain.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	return domain, nil
}

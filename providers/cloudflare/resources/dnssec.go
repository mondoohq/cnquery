// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareZone) dnssec() (*mqlCloudflareZoneDnssec, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	ds, err := conn.Cf.ZoneDNSSECSetting(context.TODO(), c.Id.Data)
	if err != nil {
		// DNSSEC may not be available on all plans (403/404)
		var notFound *cloudflare.NotFoundError
		var authN *cloudflare.AuthenticationError
		var authZ *cloudflare.AuthorizationError
		if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
			c.Dnssec.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.dnssec", map[string]*llx.RawData{
		"__id":            llx.StringData("cloudflare.zone.dnssec@" + c.Id.Data),
		"status":          llx.StringData(ds.Status),
		"flags":           llx.IntData(int64(ds.Flags)),
		"algorithm":       llx.StringData(ds.Algorithm),
		"keyType":         llx.StringData(ds.KeyType),
		"digestType":      llx.StringData(ds.DigestType),
		"digestAlgorithm": llx.StringData(ds.DigestAlgorithm),
		"digest":          llx.StringData(ds.Digest),
		"ds":              llx.StringData(ds.DS),
		"keyTag":          llx.IntData(int64(ds.KeyTag)),
		"publicKey":       llx.StringData(ds.PublicKey),
		"modifiedOn":      llx.TimeData(ds.ModifiedOn),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlCloudflareZoneDnssec), nil
}

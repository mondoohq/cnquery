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

	args := map[string]*llx.RawData{
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
	}

	res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.dnssec", args)
	if err != nil {
		return nil, err
	}

	mqlDnssec := res.(*mqlCloudflareZoneDnssec)

	// When DNSSEC is disabled the algorithm/key/digest/DS fields are not
	// meaningful — surface them as null instead of zero/empty values.
	if ds.Status == "disabled" {
		nullState := plugin.StateIsSet | plugin.StateIsNull
		mqlDnssec.Algorithm = plugin.TValue[string]{State: nullState}
		mqlDnssec.KeyType = plugin.TValue[string]{State: nullState}
		mqlDnssec.DigestType = plugin.TValue[string]{State: nullState}
		mqlDnssec.DigestAlgorithm = plugin.TValue[string]{State: nullState}
		mqlDnssec.Digest = plugin.TValue[string]{State: nullState}
		mqlDnssec.Ds = plugin.TValue[string]{State: nullState}
		mqlDnssec.KeyTag = plugin.TValue[int64]{State: nullState}
		mqlDnssec.PublicKey = plugin.TValue[string]{State: nullState}
	}

	return mqlDnssec, nil
}

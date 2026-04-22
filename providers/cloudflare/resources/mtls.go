// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareMtlsCertificate) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) mtlsCertificates() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	accountID := c.GetAccount().Data.GetId().Data

	records, _, err := conn.Cf.ListMTLSCertificates(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: accountID,
		Level:      cloudflare.AccountRouteLevel,
	}, cloudflare.ListMTLSCertificatesParams{})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range records {
		rec := records[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.mtlsCertificate", map[string]*llx.RawData{
			"id":           llx.StringData(rec.ID),
			"name":         llx.StringData(rec.Name),
			"issuer":       llx.StringData(rec.Issuer),
			"signature":    llx.StringData(rec.Signature),
			"serialNumber": llx.StringData(rec.SerialNumber),
			"ca":           llx.BoolData(rec.CA),
			"uploadedOn":   llx.TimeData(rec.UploadedOn),
			"updatedAt":    llx.TimeData(rec.UpdatedAt),
			"expiresOn":    llx.TimeData(rec.ExpiresOn),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

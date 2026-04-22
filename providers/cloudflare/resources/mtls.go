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

	acc := c.GetAccount()
	if acc.Error != nil {
		return nil, acc.Error
	}
	accountID := acc.Data.GetId().Data

	var result []any
	params := cloudflare.ListMTLSCertificatesParams{}
	for {
		records, info, err := conn.Cf.ListMTLSCertificates(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: accountID,
			Level:      cloudflare.AccountRouteLevel,
		}, params)
		if err != nil {
			return nil, err
		}

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
				"expiresAt":    llx.TimeData(rec.ExpiresOn),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !info.HasMorePages() {
			break
		}
		params.PaginationOptions.Page = info.Page + 1
	}

	return result, nil
}

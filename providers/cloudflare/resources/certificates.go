// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/cloudflare/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (c *mqlCloudflareZoneCustomCertificate) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZoneCertificatePack) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) customCertificates() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	certs, err := conn.Cf.ListSSL(context.Background(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range certs {
		cert := certs[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.customCertificate", map[string]*llx.RawData{
			"id":           llx.StringData(cert.ID),
			"hosts":        llx.ArrayData(convert.SliceAnyToInterface(cert.Hosts), types.String),
			"issuer":       llx.StringData(cert.Issuer),
			"signature":    llx.StringData(cert.Signature),
			"status":       llx.StringData(cert.Status),
			"bundleMethod": llx.StringData(cert.BundleMethod),
			"expiresAt":    llx.TimeData(cert.ExpiresOn),
			"uploadedAt":   llx.TimeData(cert.UploadedOn),
			"modifiedAt":   llx.TimeData(cert.ModifiedOn),
			"priority":     llx.IntData(cert.Priority),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

func (c *mqlCloudflareZone) certificatePacks() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	packs, err := conn.Cf.ListCertificatePacks(context.Background(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range packs {
		pack := packs[i]

		res, err := NewResource(c.MqlRuntime, "cloudflare.zone.certificatePack", map[string]*llx.RawData{
			"id":                   llx.StringData(pack.ID),
			"type":                 llx.StringData(pack.Type),
			"hosts":                llx.ArrayData(convert.SliceAnyToInterface(pack.Hosts), types.String),
			"status":               llx.StringData(pack.Status),
			"validationMethod":     llx.StringData(pack.ValidationMethod),
			"validityDays":         llx.IntData(pack.ValidityDays),
			"certificateAuthority": llx.StringData(pack.CertificateAuthority),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}

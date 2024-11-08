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

type mqlCloudflareOneInternal struct {
	ZoneID string
}

func (c *mqlCloudflareZone) one() (*mqlCloudflareOne, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.one", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.one@" + c.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	one := res.(*mqlCloudflareOne)
	one.ZoneID = c.Id.Data

	return one, nil
}

func (c *mqlCloudflareOneApp) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) apps() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}

	var result []any
	for {
		records, info, err := conn.Cf.ListAccessApplications(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.ZoneID,
			Level:      cloudflare.ZoneRouteLevel,
		}, cloudflare.ListAccessApplicationsParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			resourceData := map[string]*llx.RawData{
				"id":     llx.StringData(rec.ID),
				"aud":    llx.StringData(rec.AUD),
				"name":   llx.StringData(rec.Name),
				"domain": llx.StringData(rec.Domain),

				"allowedIdentityProviders": llx.ArrayData(convert.SliceAnyToInterface(rec.AllowedIdps), types.String),

				"appLauncherVisible":     llx.BoolData(*rec.AppLauncherVisible),
				"autoRedirectToIdentity": llx.BoolData(*rec.AutoRedirectToIdentity),
				"optionsPreflightBypass": llx.BoolData(*rec.OptionsPreflightBypass),

				"customDenyMessage":      llx.StringData(rec.CustomDenyMessage),
				"customDenyUrl":          llx.StringData(rec.CustomDenyURL),
				"serviceAuth401Redirect": llx.BoolDataPtr(rec.ServiceAuth401Redirect),

				"enableBindingCookie":     llx.BoolDataPtr(rec.EnableBindingCookie),
				"httpOnlyCookieAttribute": llx.BoolDataPtr(rec.HttpOnlyCookieAttribute),
				"sameSiteCookieAttribute": llx.StringData(rec.SameSiteCookieAttribute),

				"logoUrl":          llx.StringData(rec.LogoURL),
				"sessionDuration":  llx.StringData(rec.SessionDuration),
				"skipInterstitial": llx.BoolDataPtr(rec.SkipInterstitial),

				"type": llx.StringData(string(rec.Type)),

				"createdAt": llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt": llx.TimeDataPtr(rec.UpdatedAt),

				"corsHeaders": llx.NilData,
			}

			if rec.CorsHeaders != nil {
				headers := rec.CorsHeaders
				corsHeaders, err := NewResource(c.MqlRuntime, "cloudflare.corsHeaders", map[string]*llx.RawData{
					"allowAllHeaders":  llx.BoolData(headers.AllowAllHeaders),
					"allowAllMethods":  llx.BoolData(headers.AllowAllMethods),
					"allowAllOrigins":  llx.BoolData(headers.AllowAllOrigins),
					"allowCredentials": llx.BoolData(headers.AllowCredentials),
					"allowedHeaders":   llx.ArrayData(convert.SliceAnyToInterface(headers.AllowedHeaders), types.String),
					"allowedMethods":   llx.ArrayData(convert.SliceAnyToInterface(headers.AllowedMethods), types.String),
					"allowedOrigins":   llx.ArrayData(convert.SliceAnyToInterface(headers.AllowedOrigins), types.String),
					"maxAge":           llx.IntData(headers.MaxAge),
				})
				if err == nil {
					resourceData["corsHeaders"] = llx.ResourceData(corsHeaders, corsHeaders.MqlName())
				}
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.app", resourceData)
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

func (c *mqlCloudflareOneIdp) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) identityProviders() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}
	var result []any
	for {
		records, info, err := conn.Cf.ListAccessIdentityProviders(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.ZoneID,
			Level:      cloudflare.ZoneRouteLevel,
		}, cloudflare.ListAccessIdentityProvidersParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.idp", map[string]*llx.RawData{
				"id":   llx.StringData(rec.ID),
				"name": llx.StringData(rec.Name),
				"type": llx.StringData(string(rec.Type)),
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

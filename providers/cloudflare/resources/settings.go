// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func extractSettingStr(settings []cloudflare.ZoneSetting, id string) string {
	for _, s := range settings {
		if s.ID == id {
			if v, ok := s.Value.(string); ok {
				return v
			}
		}
	}
	return ""
}

func (c *mqlCloudflareZone) settings() (*mqlCloudflareZoneSettings, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	resp, err := conn.Cf.ZoneSettings(context.Background(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	settings := resp.Result

	res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.settings", map[string]*llx.RawData{
		"__id":                    llx.StringData("cloudflare.zone.settings@" + c.Id.Data),
		"ssl":                     llx.StringData(extractSettingStr(settings, "ssl")),
		"alwaysUseHttps":          llx.StringData(extractSettingStr(settings, "always_use_https")),
		"minTlsVersion":           llx.StringData(extractSettingStr(settings, "min_tls_version")),
		"tls13":                   llx.StringData(extractSettingStr(settings, "tls_1_3")),
		"automaticHttpsRewrites":  llx.StringData(extractSettingStr(settings, "automatic_https_rewrites")),
		"securityLevel":           llx.StringData(extractSettingStr(settings, "security_level")),
		"waf":                     llx.StringData(extractSettingStr(settings, "waf")),
		"browserCheck":            llx.StringData(extractSettingStr(settings, "browser_check")),
		"opportunisticEncryption": llx.StringData(extractSettingStr(settings, "opportunistic_encryption")),
		"emailObfuscation":        llx.StringData(extractSettingStr(settings, "email_obfuscation")),
		"hotlinkProtection":       llx.StringData(extractSettingStr(settings, "hotlink_protection")),
		"serverSideExcludes":      llx.StringData(extractSettingStr(settings, "server_side_exclude")),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlCloudflareZoneSettings), nil
}

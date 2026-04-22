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

func (c *mqlCloudflareZone) botManagement() (*mqlCloudflareZoneBotManagement, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	bm, err := conn.Cf.GetBotManagement(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.Id.Data,
	})
	if err != nil {
		// Bot management may not be available on all plans (403/404)
		var notFound *cloudflare.NotFoundError
		var authN *cloudflare.AuthenticationError
		var authZ *cloudflare.AuthorizationError
		if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
			c.BotManagement.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	res, err := CreateResource(c.MqlRuntime, "cloudflare.zone.botManagement", map[string]*llx.RawData{
		"__id":                         llx.StringData("cloudflare.zone.botManagement@" + c.Id.Data),
		"enableJs":                     llx.BoolDataPtr(bm.EnableJS),
		"fightMode":                    llx.BoolDataPtr(bm.FightMode),
		"sbfmDefinitelyAutomated":      llx.StringDataPtr(bm.SBFMDefinitelyAutomated),
		"sbfmLikelyAutomated":          llx.StringDataPtr(bm.SBFMLikelyAutomated),
		"sbfmVerifiedBots":             llx.StringDataPtr(bm.SBFMVerifiedBots),
		"sbfmStaticResourceProtection": llx.BoolDataPtr(bm.SBFMStaticResourceProtection),
		"optimizeWordpress":            llx.BoolDataPtr(bm.OptimizeWordpress),
		"autoUpdateModel":              llx.BoolDataPtr(bm.AutoUpdateModel),
		"usingLatestModel":             llx.BoolDataPtr(bm.UsingLatestModel),
		"aiBotsProtection":             llx.StringDataPtr(bm.AIBotsProtection),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlCloudflareZoneBotManagement), nil
}

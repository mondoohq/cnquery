// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package connection

import (
	"errors"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

const (
	DiscoveryAll      = "all"
	DiscoveryAuto     = "auto"
	DiscoveryZones    = "zones"
	DiscoveryAccounts = "accounts"
	// DiscoveryWorkers = "workers"
)

var CloudflareZonePlatform = inventory.Platform{
	Name:    "cloudflare-zone",
	Title:   "Cloudflare Zone",
	Family:  []string{"cloudflare"},
	Kind:    "api",
	Runtime: "cloudflare",
}

var CloudflareAccountPlatform = inventory.Platform{
	Name:    "cloudflare-account",
	Title:   "Cloudflare Account",
	Family:  []string{"cloudflare"},
	Kind:    "api",
	Runtime: "cloudflare",
}

func (c *CloudflareConnection) PlatformInfo() (*inventory.Platform, error) {
	conf := c.asset.Connections[0]
	if zoneName := conf.Options["zone"]; zoneName != "" {
		return NewCloudflareZonePlatform(zoneName), nil
	}
	if accountName := conf.Options["account"]; accountName != "" {
		return NewCloudflareAccountPlatform(accountName), nil
	}

	return nil, errors.New("could not detect Cloudflare asset type")
}

func NewCloudflareZonePlatform(zoneId string) *inventory.Platform {
	pf := CloudflareZonePlatform
	pf.TechnologyUrlSegments = []string{"saas", "cloudflare", "zone", zoneId}
	return &pf
}

func NewCloudflareAccountPlatform(accountId string) *inventory.Platform {
	pf := CloudflareAccountPlatform
	pf.TechnologyUrlSegments = []string{"saas", "cloudflare", "account", accountId}
	return &pf
}

func NewCloudflareZoneIdentifier(zoneId string) string {
	return "//platformid.api.mondoo.app/runtime/cloudflare/zone/" + zoneId
}

func NewCloudflareAccountIdentifier(accountId string) string {
	return "//platformid.api.mondoo.app/runtime/cloudflare/account/" + accountId
}

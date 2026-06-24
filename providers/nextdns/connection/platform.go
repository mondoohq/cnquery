// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

const (
	DiscoveryAll      = "all"
	DiscoveryAuto     = "auto"
	DiscoveryAccounts = "accounts"
	DiscoveryProfiles = "profiles"
)

var (
	PlatformIdNextdnsAccount = "//platformid.api.mondoo.app/runtime/nextdns/account/"
	PlatformIdNextdnsProfile = "//platformid.api.mondoo.app/runtime/nextdns/profile/"
)

var NextdnsAccountPlatform = inventory.Platform{
	Name:    "nextdns-account",
	Title:   "NextDNS Account",
	Family:  []string{"nextdns"},
	Kind:    "api",
	Runtime: "nextdns",
}

var NextdnsProfilePlatform = inventory.Platform{
	Name:    "nextdns-profile",
	Title:   "NextDNS Profile",
	Family:  []string{"nextdns"},
	Kind:    "api",
	Runtime: "nextdns",
}

// PlatformInfo returns the platform for the asset this connection is scoped to.
// A connection carrying a profile option is a single profile; otherwise it is
// the account root.
func (c *NextdnsConnection) PlatformInfo() *inventory.Platform {
	if profileID := c.ProfileID(); profileID != "" {
		return NewNextdnsProfilePlatform(profileID)
	}
	return NewNextdnsAccountPlatform(c.accountID)
}

func NewNextdnsAccountPlatform(accountID string) *inventory.Platform {
	pf := NextdnsAccountPlatform
	pf.TechnologyUrlSegments = []string{"saas", "nextdns", "account", accountID}
	return &pf
}

func NewNextdnsProfilePlatform(profileID string) *inventory.Platform {
	pf := NextdnsProfilePlatform
	pf.TechnologyUrlSegments = []string{"saas", "nextdns", "profile", profileID}
	return &pf
}

func NewNextdnsAccountIdentifier(accountID string) string {
	return PlatformIdNextdnsAccount + accountID
}

func NewNextdnsProfileIdentifier(profileID string) string {
	return PlatformIdNextdnsProfile + profileID
}

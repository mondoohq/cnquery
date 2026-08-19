// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"strconv"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Platforms is the static catalog of platforms this provider can emit.
var Platforms = []*plugin.PlatformInfo{
	{
		Name:    "newrelic",
		Title:   "New Relic Account",
		Family:  []string{"newrelic"},
		Kind:    []string{"api"},
		Runtime: []string{"newrelic"},
	},
}

var platformsByName = plugin.PlatformsByName(Platforms)

// PlatformByName returns the catalog entry for the given platform name.
func PlatformByName(name string) *plugin.PlatformInfo {
	return platformsByName[name]
}

// PlatformIdNewrelicAccount prefixes the identifier of a New Relic account
// asset.
const PlatformIdNewrelicAccount = "//platformid.api.mondoo.app/runtime/newrelic/region/"

// NewAccountIdentifier builds the platform ID of a New Relic account asset. The
// region is part of the identifier because the US and EU platforms are separate
// deployments that number their accounts independently, so an account ID alone
// is not unique across them.
func NewAccountIdentifier(region string, accountID int) string {
	return PlatformIdNewrelicAccount + region + "/account/" + strconv.Itoa(accountID)
}

// NewAccountPlatform describes a New Relic account asset.
func NewAccountPlatform(region string, accountID int) *inventory.Platform {
	pf := &inventory.Platform{
		TechnologyUrlSegments: []string{"saas", "newrelic", "region", region, "account", strconv.Itoa(accountID)},
	}
	PlatformByName("newrelic").Apply(pf)
	return pf
}

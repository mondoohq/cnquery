// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Platforms is the static catalog of platforms this provider emits. The build
// exports it into dist/stripe.json so the CLI and generated docs can list what
// the provider supports.
var Platforms = []*plugin.PlatformInfo{
	{
		Name:    "stripe",
		Title:   "Stripe Account",
		Family:  []string{"stripe"},
		Kind:    []string{"api"},
		Runtime: []string{"stripe"},
	},
}

var platformsByName = plugin.PlatformsByName(Platforms)

// PlatformByName returns the catalog entry for the given platform name.
func PlatformByName(name string) *plugin.PlatformInfo {
	return platformsByName[name]
}

// PlatformIdStripeAccount is the prefix for a Stripe account platform ID.
const PlatformIdStripeAccount = "//platformid.api.mondoo.app/runtime/stripe/account/"

// NewStripeAccountIdentifier builds the platform ID for a Stripe account asset.
func NewStripeAccountIdentifier(accountID string) string {
	return PlatformIdStripeAccount + accountID
}

// NewStripeAccountPlatform builds the platform for a Stripe account asset.
func NewStripeAccountPlatform(accountID string) *inventory.Platform {
	pf := &inventory.Platform{
		TechnologyUrlSegments: []string{"saas", "stripe", "account", accountID},
	}
	PlatformByName("stripe").Apply(pf)
	return pf
}

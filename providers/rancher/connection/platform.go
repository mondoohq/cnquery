// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/url"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// PlatformIdRancherServer prefixes the identifier of a Rancher Manager asset.
// The host is part of the identifier because the fleet is named by where its
// management plane lives, and a Rancher install has no other stable name that
// is readable before authenticating.
const PlatformIdRancherServer = "//platformid.api.mondoo.app/runtime/rancher/host/"

// NewRancherServerPlatform describes a Rancher Manager asset.
func NewRancherServerPlatform(host string) *inventory.Platform {
	segments := []string{"saas", "rancher", "host", host}

	pf := &inventory.Platform{TechnologyUrlSegments: segments}
	PlatformByName("rancher").Apply(pf)
	return pf
}

// NewRancherServerIdentifier builds the platform ID of a Rancher Manager asset.
func NewRancherServerIdentifier(host string) string {
	return PlatformIdRancherServer + url.PathEscape(host)
}

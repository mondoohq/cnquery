// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

const (
	DiscoveryAll    = "all"
	DiscoveryAuto   = "auto"
	DiscoveryRealms = "realms"
)

// PlatformIdKeycloakRealm prefixes the identifier of a realm asset. The host is
// part of the identifier because a realm name is only unique within one server,
// and the same names appear on every stage of a deployment.
const PlatformIdKeycloakRealm = "//platformid.api.mondoo.app/runtime/keycloak/host/"

func NewKeycloakRealmPlatform(host, realm string) *inventory.Platform {
	pf := &inventory.Platform{
		TechnologyUrlSegments: []string{"saas", "keycloak", "host", host, "realm", realm},
	}
	PlatformByName("keycloak-realm").Apply(pf)
	return pf
}

func NewKeycloakRealmIdentifier(host, realm string) string {
	return PlatformIdKeycloakRealm + host + "/realm/" + realm
}

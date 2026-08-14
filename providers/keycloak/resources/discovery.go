// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"maps"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/keycloak/connection"
	"go.mondoo.com/mql/v13/utils/stringx"
)

// Discover emits one asset per realm in scope, so a scan reports each realm
// separately instead of folding every realm of a server into one result.
func Discover(runtime *plugin.Runtime, opts map[string]string) (*inventory.Inventory, error) {
	conn := keycloakConn(runtime)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	targets := handleTargets(conn.Asset().Connections[0].Discover.Targets)
	if !stringx.Contains(targets, connection.DiscoveryRealms) {
		return in, nil
	}

	// A connection already scoped to one realm is that realm's asset, so
	// emitting it again would scan it twice.
	if conn.RealmFilter() != "" {
		return in, nil
	}

	root, err := getKeycloak(runtime)
	if err != nil {
		return in, err
	}

	realms := root.GetRealms()
	if realms.Error != nil {
		return in, realms.Error
	}

	conf := conn.Asset().Connections[0]
	for _, it := range realms.Data {
		realm, ok := it.(*mqlKeycloakRealm)
		if !ok {
			continue
		}

		name := realm.Name.Data
		if name == "" {
			continue
		}

		realmConf := conf.Clone()
		if realmConf.Options == nil {
			realmConf.Options = map[string]string{}
		} else {
			realmConf.Options = maps.Clone(realmConf.Options)
		}
		realmConf.Options["realmName"] = name
		realmConf.Discover = &inventory.Discovery{Targets: []string{}}

		in.Spec.Assets = append(in.Spec.Assets, &inventory.Asset{
			PlatformIds: []string{connection.NewKeycloakRealmIdentifier(conn.Host(), name)},
			Name:        conn.Host() + "/" + name,
			Platform:    connection.NewKeycloakRealmPlatform(conn.Host(), name),
			Connections: []*inventory.Config{realmConf},
		})
	}

	return in, nil
}

func handleTargets(targets []string) []string {
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryAuto) {
		return []string{connection.DiscoveryRealms}
	}
	return targets
}

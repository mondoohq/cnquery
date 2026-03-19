// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestAllResolvedResources(t *testing.T) {
	expected := []string{
		DiscoverySubscriptions,
		DiscoveryInstancesApi,
		DiscoverySqlServers,
		DiscoveryPostgresServers,
		DiscoveryPostgresFlexibleServers,
		DiscoveryMySqlServers,
		DiscoveryMySqlFlexibleServers,
		DiscoveryAksClusters,
		DiscoveryAppServiceApps,
		DiscoveryCacheRedis,
		DiscoveryBatchAccounts,
		DiscoveryStorageAccounts,
		DiscoveryKeyVaults,
		DiscoverySecurityGroups,
		DiscoveryCosmosDb,
		DiscoveryVirtualNetworks,
		DiscoveryInstances,
		DiscoveryStorageContainers,
	}
	require.ElementsMatch(t, expected, All)
}

func TestAutoResolvedResources(t *testing.T) {
	expected := []string{
		DiscoverySubscriptions,
		DiscoveryInstancesApi,
		DiscoverySqlServers,
		DiscoveryPostgresServers,
		DiscoveryPostgresFlexibleServers,
		DiscoveryMySqlServers,
		DiscoveryMySqlFlexibleServers,
		DiscoveryAksClusters,
		DiscoveryAppServiceApps,
		DiscoveryCacheRedis,
		DiscoveryBatchAccounts,
		DiscoveryStorageAccounts,
		DiscoveryKeyVaults,
		DiscoverySecurityGroups,
		DiscoveryCosmosDb,
		DiscoveryVirtualNetworks,
	}
	require.ElementsMatch(t, expected, Auto)
}

func TestGetDiscoveryTargets(t *testing.T) {
	cases := []struct {
		name    string
		targets []string
		want    []string
	}{
		{
			name:    "empty defaults to Auto",
			targets: []string{},
			want:    Auto,
		},
		{
			name:    "all",
			targets: []string{"all"},
			want:    All,
		},
		{
			name:    "all with extras",
			targets: []string{"all", "projects", "instances"},
			want:    All,
		},
		{
			name:    "auto",
			targets: []string{"auto"},
			want:    Auto,
		},
		{
			name:    "auto with extras",
			targets: []string{"auto", "postgres-servers", "keyvaults-vaults"},
			want:    append(slices.Clone(Auto), DiscoveryPostgresServers, DiscoveryKeyVaults),
		},
		{
			name:    "explicit targets",
			targets: []string{"postgres-servers", "keyvaults-vaults", "instances"},
			want:    []string{DiscoveryPostgresServers, DiscoveryKeyVaults, DiscoveryInstances},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := &inventory.Config{
				Discover: &inventory.Discovery{
					Targets: tc.targets,
				},
			}
			got := getDiscoveryTargets(config)
			require.ElementsMatch(t, tc.want, got)
		})
	}
}

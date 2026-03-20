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

func TestMinimalResolvedResources(t *testing.T) {
	expected := []string{
		DiscoverySubscriptions,
	}
	require.ElementsMatch(t, expected, Minimal)
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
			name:    "minimal",
			targets: []string{"minimal"},
			want:    Minimal,
		},
		{
			name:    "auto",
			targets: []string{"auto"},
			want:    Auto,
		},
		{
			name:    "auto with extras already in auto deduplicates",
			targets: []string{"auto", "postgres-servers", "keyvaults-vaults"},
			want:    Auto,
		},
		{
			name:    "auto with extras not in auto",
			targets: []string{"auto", "instances", "storage-containers"},
			want:    append(slices.Clone(Auto), DiscoveryInstances, DiscoveryStorageContainers),
		},
		{
			name:    "minimal and auto prefers auto",
			targets: []string{"minimal", "auto"},
			want:    Auto,
		},
		{
			name:    "minimal and all prefers all",
			targets: []string{"minimal", "all"},
			want:    All,
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

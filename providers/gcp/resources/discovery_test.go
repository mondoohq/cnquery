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
		DiscoveryOrganization,
		DiscoveryFolders,
		DiscoveryProjects,
		DiscoveryComputeImages,
		DiscoveryComputeNetworks,
		DiscoveryComputeSubnetworks,
		DiscoveryComputeFirewalls,
		DiscoveryGkeClusters,
		DiscoveryStorageBuckets,
		DiscoveryBigQueryDatasets,
		DiscoverCloudSQLMySQL,
		DiscoverCloudSQLPostgreSQL,
		DiscoverCloudSQLSQLServer,
		DiscoverCloudDNSZones,
		DiscoverCloudKMSKeyrings,
		DiscoverMemorystoreRedis,
		DiscoverMemorystoreRedisCluster,
		DiscoveryComputeInstances,
		DiscoverSecretManager,
		DiscoverPubSubTopics,
		DiscoverPubSubSubscriptions,
		DiscoverPubSubSnapshots,
		DiscoverCloudRunServices,
		DiscoverCloudRunJobs,
		DiscoverCloudFunctions,
		DiscoverDataprocClusters,
		DiscoverLoggingBuckets,
		DiscoverApiKeys,
		DiscoverIamServiceAccounts,
	}
	require.ElementsMatch(t, expected, All)
}

func TestAutoResolvedResources(t *testing.T) {
	expected := []string{
		DiscoveryOrganization,
		DiscoveryFolders,
		DiscoveryProjects,
		DiscoveryComputeImages,
		DiscoveryComputeNetworks,
		DiscoveryComputeSubnetworks,
		DiscoveryComputeFirewalls,
		DiscoveryGkeClusters,
		DiscoveryStorageBuckets,
		DiscoveryBigQueryDatasets,
		DiscoverCloudSQLMySQL,
		DiscoverCloudSQLPostgreSQL,
		DiscoverCloudSQLSQLServer,
		DiscoverCloudDNSZones,
		DiscoverCloudKMSKeyrings,
		DiscoverMemorystoreRedis,
		DiscoverMemorystoreRedisCluster,
		DiscoveryComputeInstances,
		DiscoverSecretManager,
		DiscoverPubSubTopics,
		DiscoverPubSubSubscriptions,
		DiscoverPubSubSnapshots,
		DiscoverCloudRunServices,
		DiscoverCloudRunJobs,
		DiscoverCloudFunctions,
		DiscoverDataprocClusters,
		DiscoverLoggingBuckets,
		DiscoverApiKeys,
		DiscoverIamServiceAccounts,
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
			targets: []string{"auto", "cloud-dns-zones", "compute-images"},
			want:    append(slices.Clone(Auto), DiscoverCloudDNSZones, DiscoveryComputeImages),
		},
		{
			name:    "explicit targets",
			targets: []string{"cloud-dns-zones", "compute-images", "projects", "instances"},
			want:    []string{DiscoverCloudDNSZones, DiscoveryComputeImages, DiscoveryProjects, DiscoveryComputeInstances},
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

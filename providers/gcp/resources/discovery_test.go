// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	inventoryv1 "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		DiscoverAlloyDBClusters,
		DiscoverSpannerInstances,
		DiscoverFirestoreDatabases,
		DiscoverBigtableInstances,
		DiscoverMemorystoreInstances,
		DiscoverArtifactRegistryRepos,
		DiscoverMemcacheInstances,
		DiscoverVertexAIJobs,
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
		DiscoverAlloyDBClusters,
		DiscoverSpannerInstances,
		DiscoverFirestoreDatabases,
		DiscoverBigtableInstances,
		DiscoverMemorystoreInstances,
		DiscoverArtifactRegistryRepos,
		DiscoverMemcacheInstances,
		DiscoverVertexAIJobs,
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
			name:    "empty returns nothing",
			targets: []string{},
			want:    []string{},
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
			name:    "auto with extras already in auto",
			targets: []string{"auto", "cloud-dns-zones", "compute-images"},
			want:    Auto,
		},
		{
			name:    "auto with unique extras",
			targets: []string{"auto", "some-custom-target"},
			want:    append(slices.Clone(Auto), "some-custom-target"),
		},
		{
			name:    "explicit targets",
			targets: []string{"cloud-dns-zones", "compute-images", "projects", "instances"},
			want:    []string{DiscoverCloudDNSZones, DiscoveryComputeImages, DiscoveryProjects, DiscoveryComputeInstances},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := &inventoryv1.Config{
				Discover: &inventoryv1.Discovery{
					Targets: tc.targets,
				},
			}
			got := getDiscoveryTargets(config)
			require.ElementsMatch(t, tc.want, got)
		})
	}
}

func TestIsDiscoverySkippableErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "googleapi 403", err: &googleapi.Error{Code: 403, Message: "permission denied"}, want: true},
		{name: "googleapi 404", err: &googleapi.Error{Code: 404, Message: "not found"}, want: true},
		{name: "googleapi 500", err: &googleapi.Error{Code: 500, Message: "server error"}, want: false},
		{name: "googleapi service-disabled message", err: &googleapi.Error{Code: 400, Message: "API has not been used in project 123 before or it is disabled"}, want: true},
		{name: "gRPC permission denied", err: status.Error(codes.PermissionDenied, "no access"), want: true},
		{name: "gRPC not found", err: status.Error(codes.NotFound, "missing"), want: true},
		{name: "gRPC unimplemented", err: status.Error(codes.Unimplemented, "not implemented"), want: true},
		{name: "gRPC internal", err: status.Error(codes.Internal, "boom"), want: false},
		{name: "rewrapped 403 string", err: errors.New("403: permission denied"), want: true},
		{name: "googleapi error stringified", err: errors.New("googleapi: Error 403: permission denied"), want: true},
		{name: "api-not-enabled phrasing", err: errors.New("Cloud KMS API has not been used in project foo"), want: true},
		{name: "plain unrelated error", err: errors.New("connection reset by peer"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isDiscoverySkippableErr(tc.err))
		})
	}
}

func TestRunDiscoveryStep(t *testing.T) {
	t.Run("returns nil and runs fn on success", func(t *testing.T) {
		called := false
		err := runDiscoveryStep("compute-instances", func() error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("swallows skippable errors (403)", func(t *testing.T) {
		err := runDiscoveryStep("cloud-kms-keyrings", func() error {
			return &googleapi.Error{Code: 403, Message: "permission denied"}
		})
		require.NoError(t, err, "skippable errors must not fail the whole discovery")
	})

	t.Run("swallows skippable errors (gRPC PermissionDenied)", func(t *testing.T) {
		err := runDiscoveryStep("pubsub-topics", func() error {
			return status.Error(codes.PermissionDenied, "no access")
		})
		require.NoError(t, err)
	})

	t.Run("swallows API-not-enabled errors", func(t *testing.T) {
		err := runDiscoveryStep("dataproc-clusters", func() error {
			return &googleapi.Error{Code: 403, Message: "Cloud Dataproc API has not been used in project foo before"}
		})
		require.NoError(t, err)
	})

	t.Run("propagates non-skippable errors", func(t *testing.T) {
		orig := errors.New("connection reset")
		err := runDiscoveryStep("compute-instances", func() error {
			return orig
		})
		require.ErrorIs(t, err, orig)
	})
}

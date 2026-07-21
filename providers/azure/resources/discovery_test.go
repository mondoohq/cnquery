// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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
		DiscoveryManagedHsms,
		DiscoveryIotHubs,
		DiscoverySecurityGroups,
		DiscoveryCosmosDb,
		DiscoveryVirtualNetworks,
		DiscoveryContainerRegistries,
		DiscoveryRecoveryServicesVaults,
		DiscoverySynapseWorkspaces,
		DiscoveryDataFactories,
		DiscoveryFunctionApps,
		DiscoveryApplicationGateways,
		DiscoveryFirewalls,
		DiscoveryContainerApps,
		DiscoveryCognitiveServices,
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
		DiscoveryManagedHsms,
		DiscoveryIotHubs,
		DiscoverySecurityGroups,
		DiscoveryCosmosDb,
		DiscoveryVirtualNetworks,
		DiscoveryContainerRegistries,
		DiscoveryRecoveryServicesVaults,
		DiscoverySynapseWorkspaces,
		DiscoveryDataFactories,
		DiscoveryFunctionApps,
		DiscoveryApplicationGateways,
		DiscoveryFirewalls,
		DiscoveryContainerApps,
		DiscoveryCognitiveServices,
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

func TestGetInstancesLabels(t *testing.T) {
	const vmResourceID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-rg/providers/Microsoft.Compute/virtualMachines/my-vm"

	newVM := func(props any) *mqlAzureSubscriptionComputeServiceVm {
		return &mqlAzureSubscriptionComputeServiceVm{
			Id:         plugin.TValue[string]{Data: vmResourceID, State: plugin.StateIsSet},
			Properties: plugin.TValue[any]{Data: props, State: plugin.StateIsSet},
		}
	}

	cases := []struct {
		name string
		vm   *mqlAzureSubscriptionComputeServiceVm
		want map[string]string
	}{
		{
			name: "happy path with all fields",
			vm: newVM(map[string]any{
				"vmId": "abc-123",
				"osProfile": map[string]any{
					"computerName": "host1",
				},
				"storageProfile": map[string]any{
					"osDisk": map[string]any{
						"osType": "Linux",
					},
				},
			}),
			want: map[string]string{
				"azure.mondoo.com/computername":  "host1",
				"azure.mondoo.com/ostype":        "Linux",
				"azure.mondoo.com/resourcegroup": "my-rg",
				"mondoo.com/instance":            "abc-123",
			},
		},
		{
			name: "properties not a map",
			vm:   newVM("not-a-map"),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "properties nil",
			vm:   newVM(nil),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "osProfile present but not a map",
			vm: newVM(map[string]any{
				"osProfile": "oops",
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "computerName missing",
			vm: newVM(map[string]any{
				"osProfile": map[string]any{},
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "computerName not a string",
			vm: newVM(map[string]any{
				"osProfile": map[string]any{
					"computerName": 42,
				},
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "storageProfile not a map",
			vm: newVM(map[string]any{
				"storageProfile": "nope",
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "osDisk not a map",
			vm: newVM(map[string]any{
				"storageProfile": map[string]any{
					"osDisk": 7,
				},
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "osType not a string",
			vm: newVM(map[string]any{
				"storageProfile": map[string]any{
					"osDisk": map[string]any{
						"osType": []string{"Linux"},
					},
				},
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
		{
			name: "vmId not a string",
			vm: newVM(map[string]any{
				"vmId": 12345,
			}),
			want: map[string]string{
				"azure.mondoo.com/resourcegroup": "my-rg",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getInstancesLabels(tc.vm)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetInstancesLabels_PropertiesError(t *testing.T) {
	vm := &mqlAzureSubscriptionComputeServiceVm{
		Properties: plugin.TValue[any]{Error: errors.New("boom"), State: plugin.StateIsSet},
	}
	_, err := getInstancesLabels(vm)
	require.Error(t, err)
}

func TestPropagateSubscriptionTagsToAssets(t *testing.T) {
	t.Run("fills missing keys", func(t *testing.T) {
		assets := []*inventory.Asset{{Labels: map[string]string{"a": "1"}}}
		propagateSubscriptionTagsToAssets(assets, map[string]string{"b": "2"})
		require.Equal(t, map[string]string{"a": "1", "b": "2"}, assets[0].Labels)
	})

	t.Run("asset label wins on collision", func(t *testing.T) {
		assets := []*inventory.Asset{{Labels: map[string]string{"env": "dev"}}}
		propagateSubscriptionTagsToAssets(assets, map[string]string{"env": "prod"})
		require.Equal(t, "dev", assets[0].Labels["env"])
	})

	t.Run("nil asset labels are initialized", func(t *testing.T) {
		assets := []*inventory.Asset{{}}
		propagateSubscriptionTagsToAssets(assets, map[string]string{"b": "2"})
		require.Equal(t, map[string]string{"b": "2"}, assets[0].Labels)
	})

	t.Run("empty tags is a no-op", func(t *testing.T) {
		assets := []*inventory.Asset{{Labels: map[string]string{"a": "1"}}}
		propagateSubscriptionTagsToAssets(assets, nil)
		require.Equal(t, map[string]string{"a": "1"}, assets[0].Labels)
	})

	t.Run("nil asset in slice is skipped", func(t *testing.T) {
		assets := []*inventory.Asset{nil, {Labels: map[string]string{}}}
		require.NotPanics(t, func() {
			propagateSubscriptionTagsToAssets(assets, map[string]string{"b": "2"})
		})
		require.Equal(t, map[string]string{"b": "2"}, assets[1].Labels)
	})
}

func TestAssetsForSubscription(t *testing.T) {
	assets := []*inventory.Asset{
		{Name: "a", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		{Name: "b", Labels: map[string]string{SubscriptionLabel: "sub-2"}},
		{Name: "c", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		nil,
		{Name: "d", Labels: nil},
	}
	got := assetsForSubscription(assets, "sub-1")
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].Name)
	require.Equal(t, "c", got[1].Name)
}

func TestSubToAsset_SetsSubscriptionLabel(t *testing.T) {
	asset := subToAsset(subWithConfig{
		sub: subscriptions.Subscription{
			SubscriptionID: to.Ptr("sub-1"),
			DisplayName:    to.Ptr("My Sub"),
			TenantID:       to.Ptr("tenant-1"),
		},
		conf: &inventory.Config{},
	})
	require.Equal(t, "sub-1", asset.Labels[SubscriptionLabel])
}

func TestApplySubscriptionTags_Override(t *testing.T) {
	subs := []subWithConfig{
		{sub: subscriptions.Subscription{SubscriptionID: to.Ptr("sub-1")}},
		{sub: subscriptions.Subscription{SubscriptionID: to.Ptr("sub-2")}},
	}
	assets := []*inventory.Asset{
		{Name: "vm1", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		{Name: "vm2", Labels: map[string]string{SubscriptionLabel: "sub-2", "env": "dev"}},
		{Name: "orphan", Labels: map[string]string{SubscriptionLabel: "sub-3"}},
	}

	// override wins over each subscription's own tags
	applySubscriptionTags(map[string]string{"env": "prod"}, subs, assets)

	require.Equal(t, "prod", assets[0].Labels["env"]) // filled from the override
	require.Equal(t, "dev", assets[1].Labels["env"])  // asset value wins on collision
	require.NotContains(t, assets[2].Labels, "env")   // sub-3 not in subs list — untouched
}

func TestApplySubscriptionTags_FromListedSubscription(t *testing.T) {
	// no override: tags come straight from the subscription records the list
	// pager already returned, so no extra per-subscription API call is made.
	subs := []subWithConfig{
		{sub: subscriptions.Subscription{
			SubscriptionID: to.Ptr("sub-1"),
			Tags:           map[string]*string{"owner": to.Ptr("alice"), "env": to.Ptr("prod")},
		}},
		{sub: subscriptions.Subscription{SubscriptionID: to.Ptr("sub-2")}}, // no tags
	}
	assets := []*inventory.Asset{
		{Name: "vm1", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		{Name: "vm2", Labels: map[string]string{SubscriptionLabel: "sub-1", "env": "dev"}},
		{Name: "vm3", Labels: map[string]string{SubscriptionLabel: "sub-2"}},
	}

	applySubscriptionTags(nil, subs, assets)

	require.Equal(t, "alice", assets[0].Labels["owner"]) // filled from subscription tag
	require.Equal(t, "prod", assets[0].Labels["env"])    // filled from subscription tag
	require.Equal(t, "dev", assets[1].Labels["env"])     // asset value wins on collision
	require.Equal(t, "alice", assets[1].Labels["owner"]) // still filled
	require.NotContains(t, assets[2].Labels, "owner")    // sub-2 has no tags
}

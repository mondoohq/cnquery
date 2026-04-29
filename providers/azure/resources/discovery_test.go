// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"slices"
	"testing"

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
		DiscoverySecurityGroups,
		DiscoveryCosmosDb,
		DiscoveryVirtualNetworks,
		DiscoveryContainerRegistries,
		DiscoveryRecoveryServicesVaults,
		DiscoverySynapseWorkspaces,
		DiscoveryDataFactories,
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
		DiscoveryContainerRegistries,
		DiscoveryRecoveryServicesVaults,
		DiscoverySynapseWorkspaces,
		DiscoveryDataFactories,
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
		Properties: plugin.TValue[any]{Error: assertErr("boom"), State: plugin.StateIsSet},
	}
	_, err := getInstancesLabels(vm)
	require.Error(t, err)
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

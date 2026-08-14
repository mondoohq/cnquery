// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

func TestDiscoveryStageFor(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *inventory.Config
		wantStage discoveryStage
		wantSub   string
	}{
		{
			name:      "no options at all",
			cfg:       &inventory.Config{},
			wantStage: stageLegacy,
		},
		{
			name: "client that does not set the flag stays on the legacy path",
			cfg: &inventory.Config{Options: map[string]string{
				connection.OptionSubscriptionID: "sub-1",
			}},
			wantStage: stageLegacy,
		},
		{
			name: "staged root connection walks the tenant",
			cfg: &inventory.Config{Options: map[string]string{
				plugin.OptionStagedDiscovery: "true",
			}},
			wantStage: stageTenant,
		},
		{
			// the flag is a presence check, not a truthiness check -- the k8s
			// provider sets it to the empty string
			name: "empty flag value still opts in",
			cfg: &inventory.Config{Options: map[string]string{
				plugin.OptionStagedDiscovery: "",
			}},
			wantStage: stageTenant,
		},
		{
			name: "staged connection scoped to a subscription runs stage 2",
			cfg: &inventory.Config{Options: map[string]string{
				plugin.OptionStagedDiscovery:    "true",
				connection.OptionSubscriptionID: "sub-1",
			}},
			wantStage: stageSubscription,
			wantSub:   "sub-1",
		},
		{
			name: "empty subscription id is not a scope",
			cfg: &inventory.Config{Options: map[string]string{
				plugin.OptionStagedDiscovery:    "true",
				connection.OptionSubscriptionID: "",
			}},
			wantStage: stageTenant,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stage, subId := discoveryStageFor(test.cfg)
			assert.Equal(t, test.wantStage, stage)
			assert.Equal(t, test.wantSub, subId)
		})
	}
}

func stagedRootConfig(targets ...string) *inventory.Config {
	return &inventory.Config{
		Type: "azure",
		Id:   7,
		Options: map[string]string{
			plugin.OptionStagedDiscovery: "true",
			connection.OptionTenantID:    "tenant-1",
		},
		Discover: &inventory.Discovery{Targets: targets},
	}
}

func testSubscription(id string) subscriptions.Subscription {
	return subscriptions.Subscription{
		SubscriptionID: to.Ptr(id),
		TenantID:       to.Ptr("tenant-1"),
		DisplayName:    to.Ptr("sub " + id),
	}
}

// A stage-1 subscription asset has to stay discoverable: the targets it carries
// are the only thing that makes the provider run stage 2 when the client
// connects to it. Stripping them here would end the traversal at the
// subscription and no resource would ever be discovered.
func TestTenantStageAssets_SubscriptionConfigTriggersStage2(t *testing.T) {
	rootConf := stagedRootConfig(DiscoveryAuto)
	subs := []subscriptions.Subscription{testSubscription("sub-1"), testSubscription("sub-2")}

	swcs, assets := tenantStageAssets(rootConf, subs, getDiscoveryTargets(rootConf))
	require.Len(t, assets, 2)
	require.Len(t, swcs, 2)

	for i, asset := range assets {
		require.Len(t, asset.Connections, 1)
		cfg := asset.Connections[0]

		stage, subId := discoveryStageFor(cfg)
		assert.Equal(t, stageSubscription, stage, "asset %d should route to stage 2", i)
		assert.Equal(t, *subs[i].SubscriptionID, subId)

		require.NotNil(t, cfg.Discover)
		assert.NotEmpty(t, cfg.Discover.Targets, "targets must survive so stage 2 knows what to look for")
	}
}

// The whole point of the stage boundary: a subscription must not inherit
// another connection's resource cache. Several azure resources read the
// subscription from the connection rather than from their own id, so a shared
// cache would answer with the wrong subscription rather than fail.
func TestTenantStageAssets_SubscriptionsGetTheirOwnCache(t *testing.T) {
	rootConf := stagedRootConfig(DiscoveryAuto)
	subs := []subscriptions.Subscription{testSubscription("sub-1"), testSubscription("sub-2")}

	_, assets := tenantStageAssets(rootConf, subs, getDiscoveryTargets(rootConf))

	for _, asset := range assets {
		assert.Zero(t, asset.Connections[0].ParentConnectionId,
			"a subscription must not share a resource cache with anything above it")
	}
}

func TestTenantStageAssets_ScannableWhenSubscriptionsAreTargeted(t *testing.T) {
	tests := []struct {
		name             string
		targets          []string
		wantPlatformIds  bool
		subscriptionName string
	}{
		{name: "auto includes subscriptions", targets: []string{DiscoveryAuto}, wantPlatformIds: true},
		{name: "all includes subscriptions", targets: []string{DiscoveryAll}, wantPlatformIds: true},
		{name: "explicitly targeted", targets: []string{DiscoverySubscriptions}, wantPlatformIds: true},
		{name: "mixed targets keep them scannable", targets: []string{DiscoverySubscriptions, DiscoveryInstancesApi}, wantPlatformIds: true},
		{name: "traversal only", targets: []string{DiscoveryInstancesApi}, wantPlatformIds: false},
		{name: "traversal only, several targets", targets: []string{DiscoveryInstancesApi, DiscoveryKeyVaults}, wantPlatformIds: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootConf := stagedRootConfig(test.targets...)
			subs := []subscriptions.Subscription{testSubscription("sub-1")}

			_, assets := tenantStageAssets(rootConf, subs, getDiscoveryTargets(rootConf))
			require.Len(t, assets, 1)

			if test.wantPlatformIds {
				assert.NotEmpty(t, assets[0].PlatformIds,
					"a targeted subscription must stay scannable")
			} else {
				assert.Empty(t, assets[0].PlatformIds,
					"an untargeted subscription is a traversal node, not an asset to scan")
			}
			// Either way it must still be connectable, or stage 2 never runs.
			require.Len(t, assets[0].Connections, 1)
			stage, _ := discoveryStageFor(assets[0].Connections[0])
			assert.Equal(t, stageSubscription, stage)
		})
	}
}

// Stage 2 hands every asset the subscription connection as its parent. That is
// the fix for the duplicate ARM traffic: the plugin service gives all of them
// the subscription runtime's resource cache, so a subscription-wide list is
// paid for once instead of once per asset.
func TestSubscriptionStageAssetOpts_ShareTheSubscriptionCache(t *testing.T) {
	subConf := &inventory.Config{
		Type: "azure",
		Id:   42,
		Options: map[string]string{
			plugin.OptionStagedDiscovery:    "true",
			connection.OptionSubscriptionID: "sub-1",
			connection.OptionTenantID:       "tenant-1",
		},
		Discover: &inventory.Discovery{Targets: []string{DiscoveryAuto}},
	}

	swc := subWithConfig{
		conf: subConf,
		assetOpts: []inventory.CloneOption{
			inventory.WithoutDiscovery(),
			inventory.WithParentConnectionId(subConf.Id),
		},
	}

	asset := mqlObjectToAsset(mqlObject{
		name: "vault-1",
		azureObject: azureObject{
			id:           "/subscriptions/sub-1/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/vault-1",
			subscription: "sub-1",
			tenant:       to.Ptr("tenant-1"),
			location:     "westeurope",
			service:      "keyvault",
			objectType:   "vault",
		},
	}, swc.conf, false, swc.assetOpts...)

	require.NotNil(t, asset)
	require.Len(t, asset.Connections, 1)
	cfg := asset.Connections[0]

	assert.Equal(t, uint32(42), cfg.ParentConnectionId,
		"a resource must share its subscription's resource cache")
	// A leaf that still carried discovery targets would re-run stage 2 for the
	// whole subscription, once per resource.
	assert.Empty(t, cfg.GetDiscover().GetTargets(), "resources are leaves, they discover nothing")
	stage, _ := discoveryStageFor(cfg)
	assert.Equal(t, stageSubscription, stage)
	assert.Equal(t, "sub-1", cfg.Options[connection.OptionSubscriptionID])
}

// The legacy path must keep behaving exactly as it did: no parent connection,
// no discovery on the children.
func TestLegacyAssetOpts_Unchanged(t *testing.T) {
	rootConf := &inventory.Config{
		Type:     "azure",
		Id:       9,
		Options:  map[string]string{},
		Discover: &inventory.Discovery{Targets: []string{DiscoveryAuto}},
	}
	sub := testSubscription("sub-1")

	subConf := getSubConfig(rootConf, sub, inventory.WithoutDiscovery())
	assert.Empty(t, subConf.GetDiscover().GetTargets(), "legacy sub configs do not re-discover")
	assert.Equal(t, "sub-1", subConf.Options[connection.OptionSubscriptionID])
	assert.Equal(t, "tenant-1", subConf.Options[connection.OptionTenantID])

	swc := subWithConfig{
		conf:      subConf,
		assetOpts: []inventory.CloneOption{inventory.WithoutDiscovery()},
	}
	asset := mqlObjectToAsset(mqlObject{
		name: "vault-1",
		azureObject: azureObject{
			id:           "/subscriptions/sub-1/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/vault-1",
			subscription: "sub-1",
			tenant:       to.Ptr("tenant-1"),
			location:     "westeurope",
			service:      "keyvault",
			objectType:   "vault",
		},
	}, swc.conf, false, swc.assetOpts...)

	require.NotNil(t, asset)
	require.Len(t, asset.Connections, 1)
	assert.Zero(t, asset.Connections[0].ParentConnectionId,
		"the legacy path never shared a resource cache; it must not start now")
	assert.Empty(t, asset.Connections[0].GetDiscover().GetTargets())
}

// Stage 1 writes the ids stage 2 needs into the connection options, so stage 2
// can rebuild the subscription record without asking ARM for it again.
func TestSubscriptionRecord_FromConnectionOptions(t *testing.T) {
	rootConf := stagedRootConfig(DiscoveryAuto)
	_, assets := tenantStageAssets(rootConf, []subscriptions.Subscription{testSubscription("sub-1")}, getDiscoveryTargets(rootConf))
	subConf := assets[0].Connections[0]

	conn := &connection.AzureConnection{}
	sub := subscriptionRecord(conn, subConf, "sub-1", false)

	require.NotNil(t, sub.SubscriptionID)
	assert.Equal(t, "sub-1", *sub.SubscriptionID)
	require.NotNil(t, sub.TenantID)
	assert.Equal(t, "tenant-1", *sub.TenantID)
}

func TestSubscriptionRecord_MissingTenantStaysNil(t *testing.T) {
	cfg := &inventory.Config{Options: map[string]string{
		connection.OptionSubscriptionID: "sub-1",
	}}

	sub := subscriptionRecord(&connection.AzureConnection{}, cfg, "sub-1", false)
	require.NotNil(t, sub.SubscriptionID)
	assert.Equal(t, "sub-1", *sub.SubscriptionID)
	assert.Nil(t, sub.TenantID, "an unknown tenant must stay unknown, not become the empty string")
}

// Stage 1 marks the subscription configs it has emitted assets for, so stage 2
// knows whether it has to emit the subscription itself. Reaching stage 2 from a
// command line scoped to one subscription means no stage 1 ran and nothing has
// emitted it -- the legacy path emits it either way, so this one must too.
func TestStageOneEmittedSubscription(t *testing.T) {
	rootConf := stagedRootConfig(DiscoveryAuto)
	_, assets := tenantStageAssets(rootConf, []subscriptions.Subscription{testSubscription("sub-1")}, getDiscoveryTargets(rootConf))
	require.Len(t, assets, 1)

	assert.True(t, stageOneEmittedSubscription(assets[0].Connections[0]),
		"stage 1 already emitted this subscription as an asset")

	cliScoped := &inventory.Config{Options: map[string]string{
		plugin.OptionStagedDiscovery:    "true",
		connection.OptionSubscriptionID: "sub-1",
	}}
	assert.False(t, stageOneEmittedSubscription(cliScoped),
		"a command-line scoped run never went through stage 1")
}

// Nothing beyond the ids is needed here, so the record must not be fetched.
func TestSubscriptionRecord_SkipsTheFetchWhenNothingNeedsIt(t *testing.T) {
	cfg := &inventory.Config{Options: map[string]string{
		connection.OptionSubscriptionID: "sub-1",
		connection.OptionTenantID:       "tenant-1",
	}}
	conn := &connection.AzureConnection{
		Filters: connection.DiscoveryFilters{
			PropagateSubscriptionTags: true,
			SubscriptionTags:          map[string]string{"env": "prod"},
		},
	}

	// A fetch here would panic on the nil credential, so reaching the end of
	// this call is the assertion.
	sub := subscriptionRecord(conn, cfg, "sub-1", false)
	require.NotNil(t, sub.SubscriptionID)
	assert.Equal(t, "sub-1", *sub.SubscriptionID)
	assert.Empty(t, sub.Tags)
}

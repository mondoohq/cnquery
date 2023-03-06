package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	microsoft "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Azure Compute Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{
		common.DiscoveryAuto, common.DiscoveryAll, DiscoverySubscriptions, DiscoveryInstances,
		DiscoverySqlServers, DiscoveryPostgresServers, DiscoveryMySqlServers, DiscoveryMariaDbServers,
		DiscoveryStorageAccounts, DiscoveryStorageContainers, DiscoveryKeyVaults, DiscoverySecurityGroups,
	}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	subscriptionID := tc.Options["subscription-id"]
	clientId := tc.Options["client-id"]
	subscriptionsInclude := tc.Options["subscriptions"]
	subscriptionsExclude := tc.Options["subscriptions-exclude"]

	subsToInclude := []string{}
	subsToExclude := []string{}
	if len(subscriptionID) > 0 {
		subsToInclude = append(subsToInclude, subscriptionID)
	}
	if len(subscriptionsInclude) > 0 {
		subsToInclude = append(subsToInclude, strings.Split(subscriptionsInclude, ",")...)
	}
	if len(subscriptionsExclude) > 0 {
		subsToExclude = append(subsToExclude, strings.Split(subscriptionsExclude, ",")...)
	}
	// Note: we use the resolver instead of the direct azure_provider.New to resolve credentials properly
	m, err := resolver.NewMotorConnection(ctx, tc, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	provider, ok := m.Provider.(*microsoft.Provider)
	if !ok {
		return nil, errors.New("could not create azure provider")
	}

	// if no creds, check that the CLI is installed as we are going to use that
	if clientId == "" && len(tc.Credentials) == 0 {
		azInstalled := IsAzInstalled()
		if !azInstalled {
			return nil, errors.New("az not installed")
		}
	}

	// get a token to use for discovery
	token, err := provider.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	azureClient := NewAzureClient(token)

	filter := subscriptionsFilter{
		include: subsToInclude,
		exclude: subsToExclude,
	}

	subsClient := NewSubscriptions(azureClient)
	subs, err := subsClient.GetSubscriptions(filter)
	if err != nil {
		return nil, err
	}

	type SubConfig struct {
		Cfg *providers.Config
		Sub armsubscriptions.Subscription
	}
	subsConfig := map[string]SubConfig{}
	subsAssets := map[string]*asset.Asset{}
	// we always look up subscriptions to be able to fetch their subid and tenantid to make the authentication work
	// note these are not being added as assets here, that is done only if the right targets are set down below
	for _, sub := range subs {
		// make sure we assign the correct sub and tenant id per sub, so that motor works properly
		cfg := tc.Clone()
		if cfg.Options == nil {
			cfg.Options = map[string]string{}
		}
		cfg.Options["subscription-id"] = *sub.SubscriptionID
		if cfg.Options["tenant-id"] == "" {
			cfg.Options["tenant-id"] = *sub.TenantID
		}
		subsConfig[*sub.SubscriptionID] = SubConfig{Cfg: cfg, Sub: sub}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoverySubscriptions) {
		for _, sub := range subsConfig {
			name := root.Name
			if name == "" {
				subName := *sub.Sub.SubscriptionID
				if sub.Sub.DisplayName != nil {
					subName = *sub.Sub.DisplayName
				}
				name = "Azure subscription " + subName
			}
			tc := sub.Cfg.Clone()
			p, err := microsoft.New(tc)
			if err != nil {
				return nil, err
			}

			// detect platform info for the asset
			detector := detector.New(p)
			pf, err := detector.Platform()
			if err != nil {
				return nil, err
			}
			id, _ := p.Identifier()
			subAsset := &asset.Asset{
				PlatformIds: []string{id},
				Name:        name,
				Platform:    pf,
				Connections: []*providers.Config{tc},
				Labels: map[string]string{
					"azure.com/subscription": *sub.Sub.SubscriptionID,
					"azure.com/tenant":       *sub.Sub.TenantID,
					common.ParentId:          *sub.Sub.SubscriptionID,
				},
			}
			subsAssets[*sub.Sub.SubscriptionID] = subAsset
			// if there's a root, link the sub to it
			if root != nil {
				subAsset.RelatedAssets = append(subAsset.RelatedAssets, root)
			}
			resolved = append(resolved, subAsset)
		}
	}
	// resources as assets
	// TODO: add instances here once the ip address is available viq mql
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto,
		DiscoverySqlServers, DiscoveryPostgresServers, DiscoveryMySqlServers, DiscoveryMariaDbServers,
		DiscoveryStorageAccounts, DiscoveryStorageContainers, DiscoveryKeyVaults, DiscoverySecurityGroups) {
		for id, tc := range subsConfig {
			assetList, err := GatherAssets(ctx, tc.Cfg, credsResolver, sfn)
			if err != nil {
				return nil, err
			}
			for _, a := range assetList {
				// if there's a sub available, link the asset to it
				if sub := subsAssets[id]; sub != nil {
					a.RelatedAssets = append(a.RelatedAssets, sub)
				}

				resolved = append(resolved, a)
			}
		}
	}
	// get all compute instances
	// TODO: remove this once instances are available through GatherAssets
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		for _, s := range subs {
			r := NewCompute(azureClient, *s.SubscriptionID)
			ctx := context.Background()
			assetList, err := r.ListInstances(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not fetch azure compute instances")
			}
			log.Debug().Int("instances", len(assetList)).Msg("completed instance search")

			for i := range assetList {
				a := assetList[i]

				log.Debug().Str("name", a.Name).Msg("resolved azure compute instance")
				// find the secret reference for the asset
				common.EnrichAssetWithSecrets(a, sfn)

				for i := range a.Connections {
					a.Connections[i].Insecure = tc.Insecure
				}

				// if there's a sub available, link the asset to it
				if sub := subsAssets[*s.SubscriptionID]; sub != nil {
					a.RelatedAssets = append(a.RelatedAssets, sub)
				}

				resolved = append(resolved, a)
			}
		}
	}

	return resolved, nil
}

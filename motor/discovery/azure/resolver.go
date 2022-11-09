package azure

import (
	"context"

	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	azure_provider "go.mondoo.com/cnquery/motor/providers/azure"
	"go.mondoo.com/cnquery/motor/providers/resolver"
)

const (
	DiscoverySubscriptions = "subscriptions"
	DiscoveryInstances     = "instances"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Azure Compute Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoverySubscriptions, DiscoveryInstances}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	subscriptionID := tc.Options["subscription-id"]
	clientId := tc.Options["client-id"]

	// Note: we use the resolver instead of the direct azure_provider.New to resolve credentials properly
	m, err := resolver.NewMotorConnection(ctx, tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	provider, ok := m.Provider.(*azure_provider.Provider)
	if !ok {
		return nil, errors.New("could not create azure provider")
	}

	// if no creds, check that the CLI is installed as we are going to use that
	if clientId == "" && len(tc.Credentials) == 0 {
		azInstalled := azure_provider.IsAzInstalled()
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

	// we either discover all subs (if no filter is provided) or just verify that the provided one exists
	var subs []subscriptions.Subscription
	subsClient := NewSubscriptions(azureClient)
	if subscriptionID != "" {
		subscription, err := subsClient.GetSubscription(subscriptionID)
		if err != nil {
			return nil, err
		}
		if tc.Options["tenant-id"] == "" {
			tc.Options["tenant-id"] = *subscription.TenantID
		}
		subs = []subscriptions.Subscription{subscription}
	} else {
		subs, err = subsClient.GetSubscriptions()
		if err != nil {
			return nil, err
		}
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoverySubscriptions) {
		for _, sub := range subs {
			name := root.Name
			if name == "" {
				subName := subscriptionID
				if sub.DisplayName != nil {
					subName = *sub.DisplayName
				}
				name = "Azure subscription " + subName
			}

			// make sure we assign the correct sub and tenant id per sub, so that motor works properly
			cfg := tc.Clone()
			if cfg.Options == nil {
				cfg.Options = map[string]string{}
			}
			cfg.Options["subscription-id"] = *sub.SubscriptionID
			if cfg.Options["tenant-id"] == "" {
				cfg.Options["tenant-id"] = *sub.TenantID
			}
			provider, err := azure_provider.New(cfg)
			if err != nil {
				return nil, err
			}

			// detect platform info for the asset
			detector := detector.New(provider)
			pf, err := detector.Platform()
			if err != nil {
				return nil, err
			}
			id, _ := provider.Identifier()
			resolved = append(resolved, &asset.Asset{
				PlatformIds: []string{id},
				Name:        name,
				Platform:    pf,
				Connections: []*providers.Config{cfg},
				Labels: map[string]string{
					"azure.com/subscription": *sub.SubscriptionID,
					"azure.com/tenant":       *sub.TenantID,
					common.ParentId:          *sub.SubscriptionID,
				},
			})
		}
	}

	// get all compute instances
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

				resolved = append(resolved, a)
			}
		}
	}

	return resolved, nil
}

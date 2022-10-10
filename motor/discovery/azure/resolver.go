package azure

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	azure_provider "go.mondoo.com/cnquery/motor/providers/azure"
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

	// TODO: for now we only support the azure cli authentication
	err := azure_provider.IsAzInstalled()
	if err != nil {
		return nil, err
	}

	// if we have no subscription, try to ask azure cli
	if len(subscriptionID) == 0 {
		log.Debug().Msg("no subscription id provided, fallback to azure cli")
		// read from `az account show --output json`
		account, err := azure_provider.GetAccount()
		if err == nil {
			subscriptionID = account.ID
			// NOTE: we ignore the tenant id here since we validate it below
		}
		// if an error happens, the following config validation will catch the missing subscription id
	}

	// Verify the subscription and get the details to ensure we have access
	subscription, err := azure_provider.VerifySubscription(subscriptionID)
	if err != nil || subscription.TenantID == nil {
		return nil, errors.Wrap(err, "could not fetch azure subscription details for: "+subscriptionID)
	}

	// attach tenant to config
	tc.Options["tenant-id"] = *subscription.TenantID

	provider, err := azure_provider.New(tc)
	if err != nil {
		return nil, err
	}

	identifier, err := provider.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(provider)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoverySubscriptions) {
		name := root.Name
		if name == "" {
			subName := subscriptionID
			if subscription.DisplayName != nil {
				subName = *subscription.DisplayName
			}
			name = "Azure subscription " + subName
		}

		resolved = append(resolved, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        name,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			Labels: map[string]string{
				"azure.com/subscription": subscriptionID,
				"azure.com/tenant":       *subscription.TenantID,
				common.ParentId:          subscriptionID,
			},
		})
	}

	// get all compute instances
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		r, err := NewCompute(subscriptionID)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize azure compute discovery")
		}

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

	return resolved, nil
}

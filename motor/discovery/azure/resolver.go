package azure

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	azure_transport "go.mondoo.io/mondoo/motor/transports/azure"
)

const (
	DiscoveryAll       = "all"
	DiscoveryInstances = "instances"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Azure Compute Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryInstances}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	subscriptionID := tc.Options["subscriptionID"]

	// TODO: for now we only support the azure cli authentication
	err := azure_transport.IsAzInstalled()
	if err != nil {
		return nil, err
	}

	// if we have no subscription, try to ask azure cli
	if len(subscriptionID) == 0 {
		log.Debug().Msg("no subscription id provided, fallback to azure cli")
		// read from `az account show --output json`
		account, err := azure_transport.GetAccount()
		if err == nil {
			subscriptionID = account.ID
			// NOTE: we ignore the tenant id here since we validate it below
		}
		// if an error happens, the following config validation will catch the missing subscription id
	}

	// Verify the subscription and get the details to ensure we have access
	subscription, err := azure_transport.VerifySubscription(subscriptionID)
	if err != nil || subscription.TenantID == nil {
		return nil, errors.Wrap(err, "could not fetch azure subscription details for: "+subscriptionID)
	}

	// attach tenant to config
	tc.Options["tenantID"] = *subscription.TenantID

	trans, err := azure_transport.New(tc)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	name := subscriptionID
	if subscription.DisplayName != nil {
		name = *subscription.DisplayName
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "Azure subscription " + name,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
		Labels: map[string]string{
			"azure.com/subscription": subscriptionID,
			"azure.com/tenant":       *subscription.TenantID,
			common.ParentId:          subscriptionID,
		},
	})

	// get all compute instances
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryInstances) {
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
			credentials.EnrichAssetWithSecrets(a, sfn)

			for i := range a.Connections {
				a.Connections[i].Insecure = tc.Insecure
			}

			resolved = append(resolved, a)
		}
	}

	return resolved, nil
}

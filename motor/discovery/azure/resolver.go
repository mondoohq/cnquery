package azure

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	azure_transport "go.mondoo.io/mondoo/motor/transports/azure"
)

const (
	DiscoveryAll       = "all"
	DiscoveryInstances = "instances"
)

type AzureConfig struct {
	SubscriptionID string
	User           string
}

func (az AzureConfig) Validate() error {
	if len(az.SubscriptionID) == 0 {
		return errors.New("no subscription provided, use az://subscriptions/id")
	}
	return nil
}

func parseAzureInstanceContext(azureUrl string) *AzureConfig {
	var config AzureConfig

	azureUrl = strings.TrimPrefix(azureUrl, "az://")
	azureUrl = strings.TrimPrefix(azureUrl, "azure://")

	keyValues := strings.Split(azureUrl, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == "user" {
			if i+1 < len(keyValues) {
				config.User = keyValues[i+1]
			}
		}

		if keyValues[i] == "subscriptions" {
			if i+1 < len(keyValues) {
				config.SubscriptionID = keyValues[i+1]
			}
		}

		i = i + 2
	}

	return &config
}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Azure Compute Resolver"
}

func (r *Resolver) AvailableDiscoveryModes() []string {
	return []string{DiscoveryAll, DiscoveryInstances}
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	config := parseAzureInstanceContext(url)

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	// add azure api as asset
	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_AZURE,
		Options: map[string]string{
			"subscriptionID": config.SubscriptionID,
			"user":           config.User,
		},
	}

	for i := range opts {
		opts[i](tc)
	}

	return tc, nil
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
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
		PlatformIDs: []string{identifier},
		Name:        "Azure subscription " + name,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
		Labels: map[string]string{
			"azure.com/subscription": subscriptionID,
			"azure.com/tenant":       *subscription.TenantID,
		},
	})

	// get all compute instances
	if tc.IncludesDiscovery(DiscoveryAll) || tc.IncludesDiscovery(DiscoveryInstances) {
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
			log.Debug().Str("name", assetList[i].Name).Msg("resolved azure compute instance")
			resolved = append(resolved, assetList[i])
		}
	}

	return resolved, nil
}

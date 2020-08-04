package discovery

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	azure_transport "go.mondoo.io/mondoo/motor/transports/azure"
)

type AzureConfig struct {
	Subscription  string
	ResourceGroup string
	User          string
}

func (az AzureConfig) Validate() error {
	if len(az.Subscription) == 0 {
		return errors.New("no subscription provided, use az://subscriptions/id/resourceGroups/name")
	}

	if len(az.ResourceGroup) == 0 {
		return errors.New("no resource group provided, use az://subscriptions/id/resourceGroups/name")
	}

	return nil
}

func (az AzureConfig) ResourceID() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", az.Subscription, az.ResourceGroup)
}

func ParseAzureInstanceContext(azureUrl string) *AzureConfig {
	var config AzureConfig

	azureUrl = strings.TrimPrefix(azureUrl, "az://")

	keyValues := strings.Split(azureUrl, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == "user" {
			if i+1 < len(keyValues) {
				config.User = keyValues[i+1]
			}
		}

		if keyValues[i] == "subscriptions" {
			if i+1 < len(keyValues) {
				config.Subscription = keyValues[i+1]
			}
		}

		if strings.ToLower(keyValues[i]) == "resourcegroups" {
			if i+1 < len(keyValues) {
				config.ResourceGroup = keyValues[i+1]
			}
		}

		i = i + 2
	}

	return &config
}

type azureResolver struct{}

func (k *azureResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// TODO: do not require resource groups
	// config := ParseAzureInstanceContext(in.Connection)

	// err := config.Validate()
	// if err != nil {
	// 	return nil, err
	// }

	// TODO: remove or read from `az account show --output json`
	subscriptionID := "d13c8331-50af-4496-b9d2-3ff8e72d22b9"
	tenantID := "039ce72a-f3e7-41f4-9ab4-2475a5283017"

	// add azure api as asset
	t := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_AZURE,
		Options: map[string]string{
			"subscriptionID": subscriptionID,
			"tenantID":       tenantID,
		},
	}

	trans, err := azure_transport.New(t)
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

	resolved = append(resolved, &asset.Asset{
		ReferenceIDs: []string{identifier},
		Name:         "Azure subscription " + subscriptionID,
		Platform:     pf,
		Connections:  []*transports.TransportConfig{t}, // pass-in the current config
	})

	// // get all compute instances
	// r, err := azure.NewCompute(config.ResourceID())
	// if err != nil {
	// 	return nil, errors.Wrap(err, "could not initialize azure compute discovery")
	// }

	// ctx := context.Background()
	// assetList, err := r.ListInstances(ctx)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "could not fetch azure compute instances")
	// }
	// log.Debug().Int("instances", len(assetList)).Msg("completed instance search")

	// for i := range assetList {
	// 	log.Debug().Str("name", assetList[i].Name).Msg("resolved azure compute instance")
	// 	resolved = append(resolved, assetList[i])
	// }

	return resolved, nil
}

package googleworkspace

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	google_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/motor/providers/resolver"
)

type Resolver struct{}

func (k *Resolver) Name() string {
	return "Google Workspace Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, cc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// Note: we use the resolver instead of the direct ms365_provider.New to resolve credentials properly
	m, err := resolver.NewMotorConnection(ctx, cc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	provider, ok := m.Provider.(*google_provider.Provider)
	if !ok {
		return nil, errors.New("could not create ms Google Workspace provider")
	}

	identifier, err := provider.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	if cc.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll) {
		customerID := provider.GetCustomerID()
		customer, err := provider.GetWorkspaceCustomer(customerID)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "Google Workspace " + customer.CustomerDomain,
			Platform:    pf,
			Connections: []*providers.Config{cc}, // pass-in the current config
			Labels:      map[string]string{},
		})
	}

	return resolved, nil
}

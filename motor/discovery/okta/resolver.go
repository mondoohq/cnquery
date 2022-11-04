package okta

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	okta_provider "go.mondoo.com/cnquery/motor/providers/okta"
	"go.mondoo.com/cnquery/motor/providers/resolver"
)

type Resolver struct{}

func (k *Resolver) Name() string {
	return "Okta Resolver"
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

	provider, ok := m.Provider.(*okta_provider.Provider)
	if !ok {
		return nil, errors.New("could not create ms okta provider")
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
		resolved = append(resolved, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "Okta organization " + provider.OrganizationID(),
			Platform:    pf,
			Connections: []*providers.Config{cc}, // pass-in the current config
			Labels: map[string]string{
				"okta.com/organization": provider.OrganizationID(),
			},
		})
	}

	return resolved, nil
}

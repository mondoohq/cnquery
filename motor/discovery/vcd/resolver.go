package vcd

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	vcd_provider "go.mondoo.com/cnquery/motor/providers/vcd"
)

type Resolver struct{}

func (k *Resolver) Name() string {
	return "VMware Cloud Director Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, cc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	m, err := resolver.NewMotorConnection(ctx, cc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	provider, ok := m.Provider.(*vcd_provider.Provider)
	if !ok {
		return nil, errors.New("could not create slack provider")
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
			Name:        "VMware Cloud Director", // TODO: we need to find a unique instance id
			Platform:    pf,
			Connections: []*providers.Config{cc}, // pass-in the current config
			Labels:      map[string]string{},
		})
	}

	return resolved, nil
}

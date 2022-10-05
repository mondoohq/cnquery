package equinix

import (
	"context"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	equinix_provider "go.mondoo.com/cnquery/motor/providers/equinix"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Equinix Metal Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, t *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// add aws api as asset
	provider, err := equinix_provider.New(t)
	if err != nil {
		return nil, err
	}

	identifier, err := provider.Identifier() // TODO: this identifier is not unique
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(provider)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	name := root.Name
	if name == "" {
		name = "Equinix Account" // TODO: we need to relate this to something
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        name,
		Platform:    pf,
		Connections: []*providers.Config{t}, // pass-in the current config
	})

	return resolved, nil
}

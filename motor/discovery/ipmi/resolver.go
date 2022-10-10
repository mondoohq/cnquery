package ipmi

import (
	"context"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	ipmi_provider "go.mondoo.com/cnquery/motor/providers/ipmi"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "IPMI Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, t *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	provider, err := ipmi_provider.New(t)
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

	resolved := &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        root.Name,
		Platform:    pf,
		Connections: []*providers.Config{t}, // pass-in the current config
		Labels:      map[string]string{},
	}

	// TODO: consider using the ipmi vendor id and product id
	if resolved.Name == "" {
		resolved.Name = "IPMI device " + provider.Guid()
	}

	return []*asset.Asset{resolved}, nil
}

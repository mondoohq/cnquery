package ipmi

import (
	"context"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/providers"
	ipmi_transport "go.mondoo.io/mondoo/motor/providers/ipmi"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "IPMI Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, t *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	trans, err := ipmi_transport.New(t)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(trans)
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
		resolved.Name = "IPMI device " + trans.Guid()
	}

	return []*asset.Asset{resolved}, nil
}

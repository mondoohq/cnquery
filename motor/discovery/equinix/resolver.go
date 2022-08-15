package equinix

import (
	"context"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/providers"
	equinix_transport "go.mondoo.io/mondoo/motor/providers/equinix"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Equinix Metal Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, t *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// add aws api as asset
	trans, err := equinix_transport.New(t)
	// trans, err := aws_transport.New(t, transportOpts...)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier() // TODO: this identifier is not unique
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(trans)
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

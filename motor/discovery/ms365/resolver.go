package ms365

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
	ms365_transport "go.mondoo.io/mondoo/motor/providers/ms365"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Microsoft 365 Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, cc *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// Note: we use the resolver instead of the direct ms365_transport.New to resolve credentials properly
	m, err := resolver.NewMotorConnection(ctx, cc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	trans, ok := m.Provider.(*ms365_transport.Provider)
	if !ok {
		return nil, errors.New("could not create ms 365 transport")
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "Microsoft 365 tenant " + trans.TenantID(),
		Platform:    pf,
		Connections: []*providers.Config{cc}, // pass-in the current config
		Labels: map[string]string{
			"azure.com/tenant": trans.TenantID(),
		},
	})

	return resolved, nil
}

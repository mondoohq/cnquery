package ms365

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	ms365_transport "go.mondoo.io/mondoo/motor/transports/ms365"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Microsoft 365 Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	trans, err := ms365_transport.New(t)
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
		PlatformIds: []string{identifier},
		Name:        "Microsoft 365 tenant " + trans.TenantID(),
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
		Labels: map[string]string{
			"azure.com/tenant": trans.TenantID(),
		},
	})

	return resolved, nil
}

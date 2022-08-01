package ipmi

import (
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

func (r *Resolver) Resolve(t *providers.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

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

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		// TODO: consider using the ipmi vendor id and product id
		Name:        "IPMI device " + trans.Guid(),
		Platform:    pf,
		Connections: []*providers.TransportConfig{t}, // pass-in the current config
		Labels:      map[string]string{},
	})

	return resolved, nil
}

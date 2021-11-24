package equinix

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	equinix_transport "go.mondoo.io/mondoo/motor/transports/equinix"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Equinix Metal Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(t *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
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
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "Equinix Account", // TODO: we need to relate this to something
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
	})

	return resolved, nil
}

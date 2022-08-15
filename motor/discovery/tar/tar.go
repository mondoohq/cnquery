package tar

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Tar Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	assetObj := &asset.Asset{
		Name:        root.Name,
		Connections: []*providers.Config{tc},
		State:       asset.State_STATE_ONLINE,
	}

	m, err := resolver.NewMotorConnection(ctx, tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetObj.Platform = p
	}

	fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, userIdDetectors)
	if err != nil {
		return nil, err
	}

	assetObj.PlatformIds = fingerprint.PlatformIDs
	if assetObj.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	log.Debug().Strs("identifier", assetObj.PlatformIds).Msg("motor connection")

	// use hostname as name if asset name was not explicitly provided
	if assetObj.Name == "" && tc.Options["path"] != "" {
		assetObj.Name = tc.Options["path"]
	}

	return []*asset.Asset{assetObj}, nil
}

package tar

import (
	"github.com/rs/zerolog/log"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Tar Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...string) ([]*asset.Asset, error) {
	assetInfo := &asset.Asset{
		Connections: []*transports.TransportConfig{tc},
		State:       asset.State_STATE_ONLINE,
	}

	m, err := resolver.NewMotorConnection(tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetInfo.Platform = p
	}

	platformIds, err := motorid.GatherIDs(m.Transport, p, transports.ToPlatformIdDetectors(userIdDetectors))
	if err != nil {
		return nil, err
	}
	assetInfo.PlatformIds = platformIds
	log.Debug().Strs("identifier", assetInfo.PlatformIds).Msg("motor connection")

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" && tc.Options["path"] != "" {
		assetInfo.Name = tc.Options["path"]
	}

	return []*asset.Asset{assetInfo}, nil
}

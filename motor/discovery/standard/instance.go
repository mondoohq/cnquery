package standard

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Standard Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	assetInfo := &asset.Asset{
		Connections: []*transports.TransportConfig{tc},
		State:       asset.State_STATE_ONLINE,
	}

	if len(assetInfo.Connections[0].Credentials) == 0 {
		cred, err := sfn(assetInfo)
		if err != nil {
			log.Debug().Err(err).Msg("could not determine credential for asset")
			return nil, err
		}
		assetInfo.Connections[0].Credentials = append(assetInfo.Connections[0].Credentials, cred)
	}

	m, err := resolver.NewMotorConnection(tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// store detected platform identifier with asset
	assetInfo.PlatformIds = m.Meta.Identifier
	log.Debug().Strs("identifier", assetInfo.PlatformIds).Msg("motor connection")

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetInfo.Platform = p
	}

	// use hostname as asset name
	if p != nil && assetInfo.Name == "" {
		// retrieve hostname
		hostname, err := hostname.Hostname(m.Transport, p)
		if err == nil && len(hostname) > 0 {
			assetInfo.Name = hostname
		}
	}

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" {
		assetInfo.Name = tc.Host
	}

	return []*asset.Asset{assetInfo}, nil
}

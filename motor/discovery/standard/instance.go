package standard

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/motorid"
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

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
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
		if cred != nil {
			assetInfo.Connections[0].Credentials = append(assetInfo.Connections[0].Credentials, cred)
		}
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

	platformIds, assetMetadata, err := motorid.GatherIDs(m.Transport, p, userIdDetectors)
	if err != nil {
		return nil, err
	}
	assetInfo.PlatformIds = platformIds
	if assetMetadata.Name != "" {
		assetInfo.Name = assetMetadata.Name
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

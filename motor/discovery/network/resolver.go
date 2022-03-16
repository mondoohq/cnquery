package network

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/platform/detector"
	"go.mondoo.io/mondoo/motor/transports"
	network_transport "go.mondoo.io/mondoo/motor/transports/network"
)

type Resolver struct{}

const (
	DiscoveryAll = "all"
)

func (r *Resolver) Name() string {
	return "Network Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll}
}

func (r *Resolver) Resolve(conf *transports.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	transport, err := network_transport.New(conf)
	if err != nil {
		return nil, err
	}

	detector := detector.New(transport)
	platform, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	platformID, err := transport.Identifier()
	if err != nil {
		return nil, err
	}

	assetObj := &asset.Asset{
		PlatformIds: []string{platformID},
		Platform:    platform,
		Name:        conf.Host,
		Connections: []*transports.TransportConfig{conf},
		// FIXME: We don't really know at this point if it is online... need to
		// check first
		State: asset.State_STATE_ONLINE,
	}

	return []*asset.Asset{assetObj}, nil
}

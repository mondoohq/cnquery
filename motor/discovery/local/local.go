package local

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Local Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{docker_engine.DiscoveryAll, docker_engine.DiscoveryContainerRunning, docker_engine.DiscoveryContainerImages}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	assetInfo := &asset.Asset{
		State: asset.State_STATE_ONLINE,
	}

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" {
		assetInfo.Name = tc.Host
	}

	assetInfo.Connections = []*transports.TransportConfig{tc}

	assetInfo.Platform = &platform.Platform{
		Kind: transports.Kind_KIND_BARE_METAL,
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

	platformIds, err := motorid.GatherIDs(m.Transport, p, userIdDetectors)
	if err != nil {
		return nil, err
	}
	assetInfo.PlatformIds = platformIds

	// use hostname as asset name
	if p != nil && assetInfo.Name == "" {
		// retrieve hostname
		hostname, err := hostname.Hostname(m.Transport, p)
		if err == nil && len(hostname) > 0 {
			assetInfo.Name = hostname
		}
	}
	assetList := []*asset.Asset{assetInfo}

	// search for container assets on local machine
	engineAssets, err := docker_engine.DiscoverDockerEngineAssets(tc)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, engineAssets...)

	return assetList, nil
}

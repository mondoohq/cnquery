package local

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Local Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{docker_engine.DiscoveryAll, docker_engine.DiscoveryContainerRunning, docker_engine.DiscoveryContainerImages}
}

func (r *Resolver) Resolve(root *asset.Asset, tc *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	assetObj := &asset.Asset{
		Name:        root.Name,
		State:       asset.State_STATE_ONLINE,
		Connections: []*providers.Config{tc},
	}

	// use hostname as name if asset name was not explicitly provided
	if assetObj.Name == "" {
		assetObj.Name = tc.Host
	}

	m, err := resolver.NewMotorConnection(tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetObj.Platform = p
	} else {
		assetObj.Platform = &platform.Platform{}
	}
	assetObj.Platform.Kind = providers.Kind_KIND_BARE_METAL

	fingerprint, err := motorid.IdentifyPlatform(m.Transport, p, userIdDetectors)
	if err != nil {
		return nil, err
	}

	assetObj.PlatformIds = fingerprint.PlatformIDs
	if fingerprint.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	// use hostname as asset name
	if p != nil && assetObj.Name == "" {
		// retrieve hostname
		hostname, err := hostname.Hostname(m.Transport, p)
		if err == nil && len(hostname) > 0 {
			assetObj.Name = hostname
		}
	}
	assetList := []*asset.Asset{assetObj}

	// search for container assets on local machine
	engineAssets, err := docker_engine.DiscoverDockerEngineAssets(tc)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, engineAssets...)

	return assetList, nil
}

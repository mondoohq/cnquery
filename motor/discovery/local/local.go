package local

import (
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Local Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{docker_engine.DiscoveryAll, docker_engine.DiscoveryContainerRunning, docker_engine.DiscoveryContainerImages}
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	return transports.NewTransportFromUrl(url, opts...)
}

func (r *Resolver) Resolve(tc *transports.TransportConfig) ([]*asset.Asset, error) {
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

	// this collection here is only to show the user a right indication about the asset name since -t local://
	// will lead to an empty asset name. Since the discovery process runs BEFORE the real asset collector starts,
	// we keep it intentionally lighweight, therefore we only do this for local connections
	transport, err := local.New()
	if err != nil {
		panic(err.Error())
	}

	m, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}

	p, err := m.Platform()
	if err == nil {
		// retrieve hostname
		hostname, err := hostname.Hostname(m.Transport, p)
		if err == nil && len(hostname) > 0 {
			assetInfo.Name = hostname
		}
	}

	assetList := []*asset.Asset{assetInfo}

	// search for container assets
	engineAssets, err := docker_engine.DiscoverDockerEngineAssets(tc)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, engineAssets...)

	return assetList, nil
}

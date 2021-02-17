package local

import (
	"github.com/rs/zerolog/log"
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

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	return transports.NewTransportFromUrl(url, opts...)
}

func (r *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
	assetInfo := &asset.Asset{
		State: asset.State_STATE_ONLINE,
	}

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" {
		assetInfo.Name = t.Host
	}

	assetInfo.Connections = []*transports.TransportConfig{t}

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

	// we use generic `container` and `container-images` options to avoid the requirement for the user to know if
	// the system is using docker or podman locally

	// discover running container: container:true
	if _, ok := opts["container"]; ok {
		ded, err := docker_engine.NewDockerEngineDiscovery()
		if err != nil {
			return nil, err
		}

		containerAssets, err := ded.ListContainer()
		if err != nil {
			return nil, err
		}
		log.Info().Int("images", len(containerAssets)).Msg("running container search completed")
		assetList = append(assetList, containerAssets...)
	}

	// discover container images: container-images:true
	if _, ok := opts["container-images"]; ok {
		ded, err := docker_engine.NewDockerEngineDiscovery()
		if err != nil {
			return nil, err
		}

		containerImageAssets, err := ded.ListImages()
		if err != nil {
			return nil, err
		}
		log.Info().Int("images", len(containerImageAssets)).Msg("running container image search completed")
		assetList = append(assetList, containerImageAssets...)
	}

	return assetList, nil
}

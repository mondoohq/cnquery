package instance

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Instance Resolver"
}

func (r *Resolver) AvailableDiscoveryModes() []string {
	return []string{}
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	return transports.NewTransportFromUrl(url, opts...)
}

func (r *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
	assetInfo := &asset.Asset{
		// Name: in.Name,
		// PlatformIDs: refIds,
		// Labels: in.Labels,
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

	return []*asset.Asset{assetInfo}, nil
}

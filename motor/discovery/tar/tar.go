package tar

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
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

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	filename := strings.TrimPrefix(url, "tar://")
	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_TAR,
		Options: map[string]string{
			"file": filename,
		},
	}

	for i := range opts {
		opts[i](tc)
	}

	return tc, nil
}

func (r *Resolver) Resolve(tc *transports.TransportConfig) ([]*asset.Asset, error) {
	assetInfo := &asset.Asset{
		Connections: []*transports.TransportConfig{tc},
		State:       asset.State_STATE_ONLINE,
	}

	m, err := resolver.New(tc)
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

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" && tc.Options["path"] != "" {
		assetInfo.Name = tc.Options["path"]
	}

	return []*asset.Asset{assetInfo}, nil
}

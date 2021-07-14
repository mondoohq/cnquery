package mock

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Mock Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
	return []*asset.Asset{{
		Connections: []*transports.TransportConfig{t},
	}}, nil
}

package mock

import (
	"strings"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

type mockContext struct {
	File string
}

func parseContext(connection string) mockContext {
	var config mockContext

	connection = strings.TrimPrefix(connection, "mock://")
	config.File = connection
	return config
}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Mock Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	// parse context from url
	mockCtx := parseContext(url)

	return &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_MOCK,
		Path:    mockCtx.File,
	}, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
	return []*asset.Asset{&asset.Asset{
		Connections: []*transports.TransportConfig{t},
	}}, nil
}

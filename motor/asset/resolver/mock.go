package resolver

import (
	"strings"

	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

type mockResolver struct{}

type mockContext struct {
	File string
}

func (v *mockResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// parse context from url
	context := v.ParseContext(in.Connection)
	resolved = append(resolved, mockToAsset(context, opts))
	return resolved, nil
}

func mockToAsset(mockCtx mockContext, opts *options.VulnOpts) *asset.Asset {
	return &asset.Asset{
		Connections: []*transports.TransportConfig{{
			Backend: transports.TransportBackend_CONNECTION_MOCK,
			Path:    mockCtx.File,
		}},
	}
}

func (v *mockResolver) ParseContext(connection string) mockContext {
	var config mockContext

	connection = strings.TrimPrefix(connection, "mock://")
	config.File = connection
	return config
}

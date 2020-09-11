package discovery

import (
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	ms365_transport "go.mondoo.io/mondoo/motor/transports/ms365"
)

type ms365Resolver struct{}

func (k *ms365Resolver) Name() string {
	return "Microsoft 365 Resolver"
}

func (k *ms365Resolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// add azure api as asset
	t := &transports.TransportConfig{
		Backend:       transports.TransportBackend_CONNECTION_MS365,
		Options:       map[string]string{},
		IdentityFiles: []string{in.IdentityFile},
	}

	trans, err := ms365_transport.New(t)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, &asset.Asset{
		ReferenceIDs: []string{identifier},
		Name:         "Microsoft 365 tenant " + trans.TenantID(),
		Platform:     pf,
		Connections:  []*transports.TransportConfig{t}, // pass-in the current config
		Labels: map[string]string{
			"azure.com/tenant": trans.TenantID(),
		},
	})

	return resolved, nil
}

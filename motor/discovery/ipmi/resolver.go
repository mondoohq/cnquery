package ipmi

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	ipmi_transport "go.mondoo.io/mondoo/motor/transports/ipmi"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "IPMI Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	return transports.NewTransportFromUrl(url, opts...)
	// if err != nil {
	// 	err := errors.Wrapf(err, "cannot connect to %s", url)
	// 	return nil, err
	// }

	// // copy password from opts asset if it was not encoded in url
	// if len(t.Password) == 0 && len(in.Password) > 0 {
	// 	t.Password = in.Password
	// }

	// return t, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	trans, err := ipmi_transport.New(t)
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
		PlatformIds: []string{identifier},
		// TODO: consider using the ipmi vendor id and product id
		Name:        "IPMI device " + trans.Guid(),
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
		Labels:      map[string]string{},
	})

	return resolved, nil
}

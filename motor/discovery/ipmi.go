package discovery

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	ipmi_transport "go.mondoo.io/mondoo/motor/transports/ipmi"
)

type ipmiResolver struct{}

func (k *ipmiResolver) Name() string {
	return "IPMI Resolver"
}

func (k *ipmiResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	t := &transports.TransportConfig{}
	err := t.ParseFromURI(in.Connection)
	if err != nil {
		err := errors.Wrapf(err, "cannot connect to %s", in.Connection)
		log.Error().Err(err).Msg("invalid asset connection")
	}

	// copy password from opts asset if it was not encoded in url
	if len(t.Password) == 0 && len(in.Password) > 0 {
		t.Password = in.Password
	}

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
		ReferenceIDs: []string{identifier},
		// TODO: consider using the ipmi vendor id and product id
		Name:        "IPMI device " + trans.Guid(),
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
		Labels:      map[string]string{},
	})

	return resolved, nil
}

package discovery

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type instanceResolver struct{}

func (k *instanceResolver) Name() string {
	return "Instance Resolver"
}

func (k *instanceResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {

	refIds := []string{}
	if len(in.ReferenceID) > 0 {
		refIds = []string{in.ReferenceID}
	}

	assetInfo := &asset.Asset{
		Name:         in.Name,
		ReferenceIDs: refIds,
		Labels:       in.Labels,
		State:        asset.State_STATE_ONLINE,
	}

	// parse connection from URI
	// TODO: can we avoid the convertion between asset and motor? should assets use motor connections?
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

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" {
		assetInfo.Name = t.Host
	}

	t.Sudo = &transports.Sudo{
		Active: opts.Sudo.Active,
	}

	t.IdentityFiles = []string{in.IdentityFile}
	t.Insecure = opts.Insecure
	t.BearerToken = in.BearerToken

	assetInfo.Connections = []*transports.TransportConfig{t}

	assetInfo.Platform = &platform.Platform{
		Kind: transports.Kind_KIND_BARE_METAL,
	}
	if in != nil && len(in.AssetMrn) > 0 {
		assetInfo.Mrn = in.AssetMrn
	}

	// this collection here is only to show the user a right indication about the asset name since -t local://
	// will lead to an empty asset name. Since the discovery process runs BEFORE the real asset collector starts,
	// we keep it intentionally lighweight, therefore we only do this for local connections
	if t.Backend == transports.TransportBackend_CONNECTION_LOCAL_OS {
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
	}

	return []*asset.Asset{assetInfo}, nil
}

package instance

import (
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Instance Resolver"
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	return transports.NewTransportFromUrl(url, opts...)
	// parse connection from URI
	// TODO: can we avoid the convertion between asset and motor? should assets use motor connections?
	// t, err := transports.NewTransportFromUrl(url, opts...)
	// if err != nil {
	// 	err := errors.Wrapf(err, "cannot connect to %s", url)
	// 	return nil, er
	// }

	// // copy password from opts asset if it was not encoded in url
	// if len(t.Password) == 0 && len(in.Password) > 0 {
	// 	t.Password = in.Password
	// }

	// t.Sudo = &transports.Sudo{
	// 	Active: opts.Sudo.Active,
	// }

	// t.IdentityFiles = []string{in.IdentityFile}
	// t.Insecure = opts.Insecure
	// t.BearerToken = in.BearerToken

	// return t, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
	// refIds := []string{}
	// if len(in.PlatformID) > 0 {
	// 	refIds = []string{in.PlatformID}
	// }

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
	// if in != nil && len(in.AssetMrn) > 0 {
	// 	assetInfo.Mrn = in.AssetMrn
	// }

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

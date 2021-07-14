package vsphere

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

type VMGuestResolver struct{}

func (k *VMGuestResolver) Name() string {
	return "VmWare vSphere VM Guest Resolver"
}

func (r *VMGuestResolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (k *VMGuestResolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// refIds := []string{}
	// if len(in.PlatformID) > 0 {
	// 	refIds = []string{in.PlatformID}
	// }

	assetInfo := &asset.Asset{
		// Name: in.Name,
		// PlatformIDs: refIds,
		// Labels: in.Labels,
		// TODO: we need to ask the vmware api
		State: asset.State_STATE_ONLINE,
	}

	// // parse connection from URI
	// t := &transports.TransportConfig{}
	// err := t.ParseFromURI(in.Connection)
	// if err != nil {
	// 	err := errors.Wrapf(err, "cannot connect to %s", in.Connection)
	// 	log.Error().Err(err).Msg("invalid asset connection")
	// }

	// // copy password from opts asset if it was not encoded in url
	// if len(t.Password) == 0 && len(in.Password) > 0 {
	// 	t.Password = in.Password
	// }

	// use hostname as name if asset name was not explicitly provided
	if assetInfo.Name == "" {
		assetInfo.Name = t.Host
	}

	// t.Sudo = &transports.Sudo{
	// 	Active: opts.Sudo.Active,
	// }

	// t.IdentityFiles = []string{in.IdentityFile}
	// t.Insecure = opts.Insecure

	// // add guest credentials
	// t.Options = in.Options
	// // this transport needs the following valuse
	// // t.Options = map[string]string{
	// // 	"inventoryPath": "/ha-datacenter/vm/example-centos",
	// // 	"guestUser":     "root",
	// // 	"guestPassword": "password",
	// // }

	assetInfo.Connections = []*transports.TransportConfig{t}

	// if in != nil && len(in.AssetMrn) > 0 {
	// 	assetInfo.Mrn = in.AssetMrn
	// }
	resolved = append(resolved, assetInfo)

	return resolved, nil
}

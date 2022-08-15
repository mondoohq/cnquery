package standard

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Standard Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	assetObj := &asset.Asset{
		Name:        root.Name,
		Connections: []*providers.Config{tc},
		State:       asset.State_STATE_ONLINE,
	}

	if len(assetObj.Connections[0].Credentials) == 0 {
		cred, err := sfn(assetObj)
		if err != nil {
			log.Debug().Err(err).Msg("could not determine credential for asset")
			return nil, err
		}
		if cred != nil {
			assetObj.Connections[0].Credentials = append(assetObj.Connections[0].Credentials, cred)
		}
	}

	m, err := resolver.NewMotorConnection(ctx, tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetObj.Platform = p
	}

	fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, userIdDetectors)
	if err != nil {
		return nil, err
	}
	assetObj.PlatformIds = fingerprint.PlatformIDs
	if fingerprint.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	for _, pf := range fingerprint.RelatedAssets {
		assetObj.RelatedAssets = append(assetObj.RelatedAssets, &asset.Asset{
			Name:        pf.Name,
			PlatformIds: pf.PlatformIDs,
		})
	}

	// use hostname as asset name
	if p != nil && assetObj.Name == "" {
		osProvider, isOSProvicer := m.Provider.(os.OperatingSystemProvider)
		if isOSProvicer {
			// retrieve hostname
			hostname, err := hostname.Hostname(osProvider, p)
			if err == nil && len(hostname) > 0 {
				assetObj.Name = hostname
			}
		}
	}

	// use hostname as name if asset name was not explicitly provided
	if assetObj.Name == "" {
		assetObj.Name = tc.Host
	}

	return []*asset.Asset{assetObj}, nil
}

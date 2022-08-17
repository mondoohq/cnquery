package mock

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Mock Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(root *asset.Asset, tc *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	assetObj := &asset.Asset{
		State:       asset.State_STATE_ONLINE,
		Connections: []*providers.Config{tc},
	}

	// use hostname as name if asset name was not explicitly provided
	if assetObj.Name == "" {
		assetObj.Name = tc.Host
	}

	assetObj.Platform = &platform.Platform{
		Kind: providers.Kind_KIND_BARE_METAL,
	}

	m, err := resolver.NewMotorConnection(tc, cfn)
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
	return []*asset.Asset{assetObj}, nil
}

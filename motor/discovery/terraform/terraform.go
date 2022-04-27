package terraform

import (
	"path/filepath"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Terraform Static Analysis Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	name := ""
	if tc.Options["path"] != "" {
		// manifest parent directory name
		name = common.ProjectNameFromPath(tc.Options["path"])
	}

	assetObj := &asset.Asset{
		Name:        "Terraform Static Analysis " + name,
		Connections: []*transports.TransportConfig{tc},
		State:       asset.State_STATE_ONLINE,
		Labels:      map[string]string{},
	}

	path, ok := tc.Options["path"]
	if ok {
		absPath, _ := filepath.Abs(path)
		assetObj.Labels["path"] = absPath
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

	fingerprint, err := motorid.IdentifyPlatform(m.Transport, p, userIdDetectors)
	if err != nil {
		return nil, err
	}
	assetObj.PlatformIds = fingerprint.PlatformIDs
	if fingerprint.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	return []*asset.Asset{assetObj}, nil
}

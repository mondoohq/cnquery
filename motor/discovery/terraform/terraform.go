package terraform

import (
	"path/filepath"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
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

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	name := ""
	if tc.Options["path"] != "" {
		// manifest parent directory name
		name = common.ProjectNameFromPath(tc.Options["path"])
	}

	assetInfo := &asset.Asset{
		Name:        "Terraform Static Analysis " + name,
		Connections: []*transports.TransportConfig{tc},
		State:       asset.State_STATE_ONLINE,
		Labels:      map[string]string{},
	}

	path, ok := tc.Options["path"]
	if ok {
		absPath, _ := filepath.Abs(path)
		assetInfo.Labels["path"] = absPath
	}

	m, err := resolver.NewMotorConnection(tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetInfo.Platform = p
	}

	platformIds, assetMetadata, err := motorid.GatherIDs(m.Transport, p, userIdDetectors)
	if err != nil {
		return nil, err
	}
	assetInfo.PlatformIds = platformIds
	if assetMetadata.Name != "" {
		assetInfo.Name = assetMetadata.Name
	}

	return []*asset.Asset{assetInfo}, nil
}
